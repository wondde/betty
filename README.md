# Betty

BerryTicket을 macOS 메뉴바에서 쓰는 앱입니다. 나에게 배정된 업무를 급한 순서로 보여 주고, 어디서든 단축키로 업무를 추가할 수 있습니다.

- **메뉴바 창**: 베리 아이콘을 누르면 할 일 목록이 아이콘 아래에 열립니다. 아이콘 옆 숫자는 남은 업무 수입니다. Dock에는 나타나지 않습니다.
- **빠른 추가 (⌥⌘N)**: 어느 앱, 어느 Space에서든 Spotlight처럼 입력창이 뜹니다.
  `@민수 배너 문구 정리 #나루 금요일까지` → 담당자, 프로젝트, 마감일까지 한 번에 등록되고, 이름과 프로젝트는 자동완성됩니다.
- **업무 카드**: 동그라미를 누르면 완료됩니다(1.6초 안에 다시 누르면 취소). ▶/❚❚로 시작하거나 멈추고, 카드를 누르면 설명과 막힘, 원문, 보드 링크가 나옵니다.
- **정렬**: 지난 마감 → 오늘/내일 마감 → 새로 들어온 일 → 진행 중 → 막힘 → 할 일 순서입니다.
- **알림**: 새로 배정된 업무는 macOS 알림으로 알려 줍니다. 알림을 누르면 목록이 열립니다.
- **로그인**: Discord 로그인 후 기기 토큰을 **키체인**에 저장합니다. 메뉴의 로그아웃을 누르면 서버에서도 이 기기가 해제됩니다.
- **우클릭 메뉴**: 빠른 추가, 웹 보드 열기, 새로고침, 로그인 시 자동 실행, 로그아웃, 종료.

## 구조

```
main.go                 창 2개(메뉴바 목록, 빠른 추가 패널), 메뉴바 아이콘, 전역 단축키, 단일 실행
betty.go                프런트엔드가 부르는 유일한 서비스: 동기화(15초), 알림, 로그인, 업무 변경
internal/berry          BerryTicket 위젯 API 클라이언트 (/api/widget/*, /api/devices/*)
internal/signin         Discord 로그인: 시스템 브라우저 + 127.0.0.1 루프백 콜백 (RFC 8252) + PKCE식 검증
internal/vault          키체인에 기기 토큰 보관
internal/prefs          ~/Library/Application Support/Betty/prefs.json: 확인한 업무, 오프라인 스냅샷
frontend/src            Svelte 5: TaskWindow(목록), QuickAdd(빠른 추가), Composer(@ # 날짜 입력)
assets/tray-template.png  메뉴바 템플릿 아이콘 (go run ./tools/trayicon 으로 다시 생성)
```

Go 서비스가 바뀌면 `betty:view` 이벤트로 모든 창에 같은 상태가 전달됩니다. 네트워크와 키체인 작업은 전부 Go 쪽에서 하고, 화면은 그리기만 합니다.

## 개발

필요한 도구: Go 1.26+, Node 20+, Xcode Command Line Tools, Wails v3 CLI(`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`)

```sh
wails3 dev                      # 개발 모드 (핫 리로드)
wails3 task darwin:package      # bin/betty.app
go test ./internal/...          # 로그인 흐름, API 클라이언트 테스트
BETTY_SERVER=http://127.0.0.1:8000 open bin/betty.app   # 로컬 BerryTicket 서버에 연결
```

서버 측 요구사항: BerryTicket에 Discord 로그인, 기기 토큰, 위젯 API, 루프백 리디렉션(`redirect=http://127.0.0.1:<port>/callback`)이 배포되어 있어야 합니다.

## 참고

oneul 위젯(Lanafksk/oneul)의 형태를 참고했지만 코드는 새로 작성했습니다.
