// Betty — BerryTicket for macOS.
//
// A regular Mac app: Dock icon, the usual app menu (Betty · 파일 · 편집 · 보기 · 윈도우 · 도움말)
// and a normal titled window with my tasks. Extras on top of that:
//
//	⌥⌘N (global)    Spotlight-style quick add panel (@name #project 내일)
//	⌥⌘B (global)    bring the task window forward from anywhere
//	menu bar icon   the same window, plus quick add / board in its menu
//
// All network, Keychain and notification work is in the Betty service (betty.go);
// menu commands that need the UI are sent to it as "betty:command" events.
package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/wondde/betty/internal/prefs"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/tray-template.png
var trayIcon []byte

const (
	quickAddShortcut  = "CmdOrCtrl+Alt+N"
	showTasksShortcut = "CmdOrCtrl+Alt+B"
)

// A development server (BETTY_SERVER) runs as its own instance next to the real Betty.
func instanceID() string {
	if os.Getenv("BETTY_SERVER") != "" {
		return "com.blueberryteam.betty.dev"
	}
	return "com.blueberryteam.betty"
}

func init() {
	application.RegisterEvent[View]("betty:view")
	application.RegisterEvent[string]("betty:quick-open")
	// about | new-task | search | settings | onboarding | toggle-done | shortcuts | go-<view>
	application.RegisterEvent[string]("betty:command")
}

func main() {
	ns := notifications.New()
	betty := NewBetty(prefs.Open(), ns)

	var showTasks func()
	app := application.New(application.Options{
		Name:        "Betty",
		Description: "BerryTicket for macOS",
		Services: []application.Service{
			application.NewService(betty),
			application.NewService(ns),
		},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyRegular,
			// Closing the window keeps Betty running for notifications; quit with ⌘Q.
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: instanceID(),
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if showTasks != nil {
					showTasks()
				}
			},
		},
	})

	// ---------- main window ----------
	tasks := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "Betty",
		Width:     1040,
		Height:    700,
		MinWidth:  760,
		MinHeight: 480,
		Mac: application.MacWindow{
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
			InvisibleTitleBarHeight: 52,
		},
		BackgroundType: application.BackgroundTypeTranslucent,
		URL:            "/",
	})
	tasks.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		tasks.Hide()
		e.Cancel()
	})
	showTasks = func() {
		tasks.Show()
		tasks.Focus()
	}
	betty.onShowTasks = showTasks
	command := func(name string) {
		showTasks()
		tasks.EmitEvent("betty:command", name)
	}
	// Clicking the Dock icon brings the window back after it was closed.
	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) { showTasks() })

	// ---------- quick add panel ----------
	quick := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            "quick",
		Title:           "Betty 빠른 추가",
		Width:           620,
		Height:          64,
		Frameless:       true,
		Hidden:          true,
		DisableResize:   true,
		HideOnEscape:    true,
		HideOnFocusLost: true,
		InitialPosition: application.WindowCentered,
		BackgroundType:  application.BackgroundTypeTransparent,
		Mac: application.MacWindow{
			Backdrop:    application.MacBackdropLiquidGlass,
			LiquidGlass: application.MacLiquidGlass{Style: application.LiquidGlassStyleAutomatic, Material: application.NSVisualEffectMaterialAuto, CornerRadius: 16},
			TitleBar:    application.MacTitleBar{AppearsTransparent: true, Hide: true},
			WindowClass: application.MacWindowClassPanel,
			WindowLevel: application.MacWindowLevelFloating,
			// Shows over full-screen apps on every Space, like Spotlight.
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces | application.MacWindowCollectionBehaviorFullScreenAuxiliary,
		},
		URL: "/#/quick",
	})
	quick.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		quick.Hide()
		e.Cancel()
	})
	openQuick := func() {
		quick.Center()
		quick.Show()
		quick.Focus()
		quick.EmitEvent("betty:quick-open", "")
	}

	// ---------- app menu (standard macOS layout, Korean labels) ----------
	item := func(label string, base *application.MenuItem) *application.MenuItem { return base.SetLabel(label) }
	menu := app.NewMenu()

	appMenu := menu.AddSubmenu("Betty")
	appMenu.Add("Betty에 관하여").OnClick(func(*application.Context) { command("about") })
	appMenu.AddSeparator()
	appMenu.Add("설정…").SetAccelerator("CmdOrCtrl+,").OnClick(func(*application.Context) { command("settings") })
	appMenu.AddSeparator()
	appMenu.Add("로그아웃").OnClick(func(*application.Context) { _ = betty.SignOut() })
	appMenu.AddSeparator()
	appMenu.Append(application.NewMenuFromItems(
		item("Betty 가리기", application.NewHideMenuItem()),
		item("기타 가리기", application.NewHideOthersMenuItem()),
		item("모두 보기", application.NewShowAllMenuItem()),
	))
	appMenu.AddSeparator()
	appMenu.Append(application.NewMenuFromItems(item("Betty 종료", application.NewQuitMenuItem())))

	fileMenu := menu.AddSubmenu("파일")
	fileMenu.Add("새 할 일").SetAccelerator("CmdOrCtrl+N").OnClick(func(*application.Context) { command("new-task") })
	fileMenu.Add("빠른 추가").SetAccelerator(quickAddShortcut).OnClick(func(*application.Context) { openQuick() })
	fileMenu.Add("대화로 할 일 만들기…").SetAccelerator("CmdOrCtrl+Shift+N").OnClick(func(*application.Context) { command("capture") })
	fileMenu.AddSeparator()
	fileMenu.Add("새로고침").SetAccelerator("CmdOrCtrl+R").OnClick(func(*application.Context) { go betty.Refresh() })
	fileMenu.Add("웹 보드 열기").SetAccelerator("CmdOrCtrl+Shift+B").OnClick(func(*application.Context) { _ = betty.OpenBoard() })
	fileMenu.AddSeparator()
	fileMenu.Append(application.NewMenuFromItems(item("창 닫기", application.NewCloseMenuItem())))

	editMenu := menu.AddSubmenu("편집")
	editMenu.Append(application.NewMenuFromItems(
		item("실행 취소", application.NewUndoMenuItem()),
		item("실행 복귀", application.NewRedoMenuItem()),
	))
	editMenu.AddSeparator()
	editMenu.Append(application.NewMenuFromItems(
		item("오려두기", application.NewCutMenuItem()),
		item("복사하기", application.NewCopyMenuItem()),
		item("붙여넣기", application.NewPasteMenuItem()),
		item("전체 선택", application.NewSelectAllMenuItem()),
	))
	editMenu.AddSeparator()
	editMenu.Add("찾기").SetAccelerator("CmdOrCtrl+F").OnClick(func(*application.Context) { command("search") })

	viewMenu := menu.AddSubmenu("보기")
	for _, v := range []struct{ label, key, cmd string }{
		{"내 할 일", "CmdOrCtrl+1", "go-mine"},
		{"팀 할 일", "CmdOrCtrl+2", "go-team"},
		{"베티가 찾은 일", "CmdOrCtrl+3", "go-inbox"},
		{"캘린더", "CmdOrCtrl+4", "go-calendar"},
		{"목록/보드 전환", "CmdOrCtrl+B", "toggle-board"},
	} {
		cmd := v.cmd
		viewMenu.Add(v.label).SetAccelerator(v.key).OnClick(func(*application.Context) { command(cmd) })
	}
	viewMenu.AddSeparator()
	viewMenu.Add("최근 완료 보기/숨기기").SetAccelerator("CmdOrCtrl+Shift+D").OnClick(func(*application.Context) { command("toggle-done") })

	windowMenu := menu.AddSubmenu("윈도우")
	windowMenu.Append(application.NewMenuFromItems(
		item("최소화", application.NewMinimiseMenuItem()),
		item("확대/축소", application.NewZoomMenuItem()),
	))
	windowMenu.AddSeparator()
	windowMenu.Add("할 일 창").SetAccelerator(showTasksShortcut).OnClick(func(*application.Context) { showTasks() })

	helpMenu := menu.AddSubmenu("도움말")
	helpMenu.Add("Betty 사용법 다시 보기").OnClick(func(*application.Context) { command("onboarding") })
	helpMenu.Add("키보드 단축키").OnClick(func(*application.Context) { command("shortcuts") })

	app.Menu.SetApplicationMenu(menu)

	// ---------- menu bar icon ----------
	tray := app.SystemTray.New()
	tray.SetTemplateIcon(trayIcon)
	tray.SetTooltip("Betty")
	tray.OnClick(showTasks)
	trayMenu := app.NewMenu()
	trayMenu.Add("할 일 창 열기").OnClick(func(*application.Context) { showTasks() })
	trayMenu.Add("빠른 추가").SetAccelerator(quickAddShortcut).OnClick(func(*application.Context) { openQuick() })
	trayMenu.Add("웹 보드 열기").OnClick(func(*application.Context) { _ = betty.OpenBoard() })
	trayMenu.AddSeparator()
	trayMenu.Add("Betty 종료").OnClick(func(*application.Context) { app.Quit() })
	tray.SetMenu(trayMenu)
	betty.onCount = func(open int) {
		label := ""
		if open > 0 {
			label = fmt.Sprint(open)
		}
		tray.SetLabel(label)
	}

	// ---------- global shortcuts ----------
	if err := app.GlobalShortcut.Register(quickAddShortcut, openQuick); err != nil {
		log.Printf("quick add shortcut %s unavailable: %v", quickAddShortcut, err)
	}
	if err := app.GlobalShortcut.Register(showTasksShortcut, showTasks); err != nil {
		log.Printf("task window shortcut %s unavailable: %v", showTasksShortcut, err)
	}

	// Clicking a "new task" notification opens the task list.
	ns.OnNotificationResponse(func(notifications.NotificationResult) { showTasks() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
