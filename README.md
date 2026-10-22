# Betty

BerryTicket을 위한 macOS 앱입니다. 나에게 배정된 업무를 급한 순서로 보여 주고, 어디서든 단축키로 업무를 추가할 수 있습니다.

- **일반 맥 앱**: Dock 아이콘과 앱 메뉴(Betty · 파일 · 편집 · 보기 · 윈도우 · 도움말)가 있습니다. 창을 닫아도 Betty는 계속 실행되면서 알림을 받고, Dock 아이콘을 누르면 창이 다시 열립니다. 종료는 ⌘Q입니다.
- **처음 로그인하면 사용법 안내**가 5단계로 나오고, 마지막 단계에서 알림과 로그인 시 자동 실행을 켤 수 있습니다. 도움말 → "Betty 사용법 다시 보기"로 다시 볼 수 있습니다.
- **업무 카드**: 동그라미를 누르면 완료됩니다(1.6초 안에 다시 누르면 취소). ▶/❚❚로 시작하거나 멈추고, 카드를 누르면 설명과 막힘, 원문, 보드 링크가 나옵니다.
- **정렬**: 마감 임박 → 새로 들어온 일 → 진행 중 → 막힘 → 할 일 순서로 묶어서 보여 줍니다.
- **할 일 추가**: 창 아래 입력창 또는 ⌘N. `@민수 배너 문구 정리 #나루 금요일까지`처럼 쓰면 담당자, 프로젝트, 마감일이 한 번에 정해지고, 이름과 프로젝트는 자동완성됩니다.
- **어디서든**: ⌥⌘N 빠른 추가(Spotlight처럼 화면 가운데에 뜸), ⌥⌘B Betty 창 열기.
- **설정 (⌘,)**: 계정과 로그아웃, 알림, 로그인 시 자동 실행, 사용법 다시 보기. 도움말 → 키보드 단축키에서 전체 단축키를 볼 수 있습니다.
- **메뉴바 아이콘**: 보조 수단입니다. 누르면 Betty 창이 열리고, 옆 숫자는 남은 업무 수입니다(노치에 가려질 수 있어요).
- **로그인**: Discord 로그인 후 기기 토큰을 **키체인**에 저장합니다. 로그아웃하면 서버에서도 이 기기가 해제됩니다.

## 구조

```
main.go                 앱 메뉴, 메인 창, 빠른 추가 패널, 메뉴바 아이콘, 전역 단축키, 단일 실행
betty.go                프런트엔드가 부르는 유일한 서비스: 동기화(15초), 알림, 로그인, 업무 변경
internal/berry          BerryTicket 위젯 API 클라이언트 (/api/widget/*, /api/devices/*)
internal/signin         Discord 로그인: 시스템 브라우저 + 127.0.0.1 루프백 콜백 (RFC 8252) + PKCE식 검증
internal/vault          키체인에 기기 토큰 보관
internal/prefs          ~/Library/Application Support/Betty/prefs.json: 확인한 업무, 오프라인 스냅샷
frontend/src            Svelte 5: TaskWindow(목록), Onboarding(첫 안내), Sheet(설정·단축키·정보), QuickAdd, Composer(@ # 날짜 입력)
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
