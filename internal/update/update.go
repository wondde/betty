// Package update keeps the Mac app current with one click, the way chat apps do it.
//
// The server publishes /download/latest.json ({version, zip, sha256, notes}) next to the
// notarized app as a zip. Check compares versions; Install downloads the zip, checks the
// SHA-256 from latest.json, unpacks it, refuses anything not signed by our Developer ID team
// with the same bundle id, swaps it in for the running app and relaunches it.
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TeamID and BundleID an update must be signed with.
const (
	TeamID   = "4JAB44S28T"
	BundleID = "com.blueberryteam.betty"
)

// Info is a newer version the server offers.
type Info struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	Zip     string `json:"zip"`    // file name next to latest.json
	SHA256  string `json:"sha256"` // of the zip
}

var client = &http.Client{Timeout: 10 * time.Minute}

// Check returns the offered version when it is newer than current, else nil.
func Check(ctx context.Context, server, current string) (*Info, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server+"/download/latest.json", nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("업데이트 확인 실패 (HTTP %d)", res.StatusCode)
	}
	var info Info
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<16)).Decode(&info); err != nil {
		return nil, err
	}
	if !Newer(info.Version, current) {
		return nil, nil
	}
	return &info, nil
}

// Newer reports whether version a is later than b (dotted numbers; anything else is not newer).
func Newer(a, b string) bool {
	pa, pb := parts(a), parts(b)
	if pa == nil || pb == nil {
		return false
	}
	for i := 0; i < len(pa) || i < len(pb); i++ {
		x, y := at(pa, i), at(pb, i)
		if x != y {
			return x > y
		}
	}
	return false
}

func parts(v string) []int {
	var out []int
	for _, p := range strings.Split(strings.TrimPrefix(v, "v"), ".") {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil
		}
		out = append(out, n)
	}
	return out
}

func at(p []int, i int) int {
	if i < len(p) {
		return p[i]
	}
	return 0
}

// Bundle is the .app the running executable lives in ("" when not running from a bundle,
// e.g. `wails3 dev`).
func Bundle() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exe, _ = filepath.EvalSymlinks(exe)
	app := filepath.Dir(filepath.Dir(filepath.Dir(exe))) // …/Betty.app/Contents/MacOS/betty
	if !strings.HasSuffix(app, ".app") {
		return ""
	}
	return app
}

// ErrNotWritable means the app's folder can't be changed by this user; download it instead.
var ErrNotWritable = errors.New("이 맥에서는 앱을 직접 바꿀 수 없어요. 다운로드 페이지에서 받아 주세요")

// Install downloads, verifies and swaps in the update, then relaunches. On success the caller
// should quit right away.
func Install(ctx context.Context, server string, info Info) error {
	app := Bundle()
	if app == "" {
		return errors.New("개발용 실행에서는 업데이트하지 않아요")
	}
	if !writable(filepath.Dir(app)) || !writable(app) {
		return ErrNotWritable
	}
	if strings.ContainsAny(info.Zip, "/\\") || !strings.HasSuffix(info.Zip, ".zip") || len(info.SHA256) != 64 {
		return errors.New("업데이트 정보가 올바르지 않아요")
	}
	// Work next to the app so the swap is a rename on the same volume.
	work, err := os.MkdirTemp(filepath.Dir(app), ".betty-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	zip := filepath.Join(work, info.Zip)
	if err := download(ctx, server+"/download/"+info.Zip, zip, info.SHA256); err != nil {
		return err
	}
	unpacked := filepath.Join(work, "new")
	if out, err := exec.CommandContext(ctx, "/usr/bin/ditto", "-x", "-k", zip, unpacked).CombinedOutput(); err != nil {
		return fmt.Errorf("압축을 풀지 못했어요: %s", strings.TrimSpace(string(out)))
	}
	apps, _ := filepath.Glob(filepath.Join(unpacked, "*.app"))
	if len(apps) != 1 {
		return errors.New("업데이트 파일에 앱이 없어요")
	}
	if err := verify(ctx, apps[0]); err != nil {
		return err
	}

	// Swap: move the running app aside (it keeps running from its new place), move the new one
	// in, and bring the old one back if that fails.
	old := filepath.Join(work, "old.app")
	if err := os.Rename(app, old); err != nil {
		return fmt.Errorf("기존 앱을 옮기지 못했어요: %w", err)
	}
	if err := os.Rename(apps[0], app); err != nil {
		if out, cerr := exec.Command("/usr/bin/ditto", apps[0], app).CombinedOutput(); cerr != nil {
			_ = os.RemoveAll(app)
			_ = os.Rename(old, app)
			return fmt.Errorf("새 앱을 넣지 못했어요: %s", strings.TrimSpace(string(out)))
		}
	}
	// Relaunch once this process has quit (open -n would start a second copy right away and
	// the single-instance lock would hand it back to us).
	script := fmt.Sprintf("while kill -0 %d 2>/dev/null; do sleep 0.2; done; /usr/bin/open %q", os.Getpid(), app)
	return exec.Command("/bin/sh", "-c", script).Start()
}

func download(ctx context.Context, url, dest, want string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("내려받지 못했어요: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("내려받지 못했어요 (HTTP %d)", res.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(f, h), io.LimitReader(res.Body, 500<<20))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("내려받지 못했어요: %w", err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(got, want) {
		return errors.New("내려받은 파일이 서버의 정보와 달라요. 설치하지 않았어요")
	}
	return nil
}

// verify accepts only a validly signed app from our Developer ID team with our bundle id.
func verify(ctx context.Context, app string) error {
	if out, err := exec.CommandContext(ctx, "/usr/bin/codesign", "--verify", "--deep", "--strict", app).CombinedOutput(); err != nil {
		return fmt.Errorf("서명을 확인하지 못했어요: %s", strings.TrimSpace(string(out)))
	}
	out, _ := exec.CommandContext(ctx, "/usr/bin/codesign", "-dvv", app).CombinedOutput() // -dvv lists the signing authorities
	info := string(out)
	if !strings.Contains(info, "TeamIdentifier="+TeamID+"\n") || !strings.Contains(info, "Identifier="+BundleID+"\n") || !strings.Contains(info, "Authority=Developer ID Application") {
		return errors.New("우리 팀이 서명한 앱이 아니라 설치하지 않았어요")
	}
	return nil
}

func writable(path string) bool {
	f, err := os.CreateTemp(path, ".betty-write-test-")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}
