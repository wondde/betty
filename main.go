// Betty — BerryTicket in the macOS menu bar.
//
//	menu bar icon ── click ──▶ "main" popover window (my tasks)
//	⌥⌘B (global) / reopening Betty.app ─▶ the same window, centred on screen
//	⌥⌘N (global)  ──────────▶ "quick" Spotlight-style panel (add a task, @name #project 내일)
//	right click   ──────────▶ menu: quick add, web board, launch at login, sign out, quit
//
// On notched MacBooks a crowded menu bar hides icons behind the notch, so nothing may
// depend on the menu bar icon alone: the window also opens centred from the shortcut,
// from relaunching the app, and on first launch while signed out.
//
// All network, Keychain and notification work is in the Betty service (betty.go).
package main

import (
	"embed"
	"fmt"
	"log"

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

func init() {
	application.RegisterEvent[View]("betty:view")
	application.RegisterEvent[string]("betty:quick-open")
}

func main() {
	ns := notifications.New()
	betty := NewBetty(prefs.Open(), ns)

	var tray *application.SystemTray
	var showCentered func()
	app := application.New(application.Options{
		Name:        "Betty",
		Description: "BerryTicket in the menu bar",
		Services: []application.Service{
			application.NewService(betty),
			application.NewService(ns),
		},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac: application.MacOptions{
			// Menu bar app: no Dock icon, no app switcher entry.
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.blueberryteam.betty",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if showCentered != nil {
					showCentered()
				}
			},
		},
	})

	glass := application.MacWindow{
		Backdrop:    application.MacBackdropLiquidGlass,
		LiquidGlass: application.MacLiquidGlass{Style: application.LiquidGlassStyleAutomatic, Material: application.NSVisualEffectMaterialAuto, CornerRadius: 18},
		TitleBar:    application.MacTitleBar{AppearsTransparent: true, Hide: true},
	}

	// My tasks, dropped down from the menu bar icon.
	tasks := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            "main",
		Title:           "Betty",
		Width:           368,
		Height:          560,
		Frameless:       true,
		Hidden:          true,
		DisableResize:  true,
		HideOnEscape:   true,
		AlwaysOnTop:    true,
		BackgroundType: application.BackgroundTypeTransparent,
		Mac:            glass,
		URL:            "/",
	})
	tasks.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		tasks.Hide()
		e.Cancel()
	})
	// Dropped down from the menu bar it behaves like a popover and hides when you click
	// elsewhere. Opened centred (shortcut, relaunch, sign-in) it stays until closed: a
	// background app may not get focus right away, and hiding on that would make it vanish.
	var fromTray bool
	tasks.OnWindowEvent(events.Common.WindowLostFocus, func(*application.WindowEvent) {
		if fromTray {
			tasks.Hide()
		}
	})
	showCentered = func() {
		fromTray = false
		tasks.Center()
		tasks.Show()
		tasks.Focus()
	}
	toggleCentered := func() {
		if tasks.IsVisible() {
			tasks.Hide()
		} else {
			showCentered()
		}
	}
	betty.onShowTasks = showCentered

	// Quick add, like Spotlight: floats over full-screen apps on every Space.
	quickGlass := glass
	quickGlass.WindowClass = application.MacWindowClassPanel
	quickGlass.WindowLevel = application.MacWindowLevelFloating
	quickGlass.CollectionBehavior = application.MacWindowCollectionBehaviorCanJoinAllSpaces | application.MacWindowCollectionBehaviorFullScreenAuxiliary
	quickGlass.LiquidGlass.CornerRadius = 16
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
		Mac:             quickGlass,
		URL:             "/#/quick",
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

	// Menu bar icon with the number of open tasks next to it.
	tray = app.SystemTray.New()
	tray.SetTemplateIcon(trayIcon)
	tray.SetTooltip("Betty")
	tray.AttachWindow(tasks).WindowOffset(6)
	tray.OnClick(func() {
		fromTray = true
		tray.ToggleWindow()
	})
	betty.onCount = func(open int) {
		if open > 0 {
			tray.SetLabel(fmt.Sprint(open))
		} else {
			tray.SetLabel("")
		}
	}

	menu := app.NewMenu()
	menu.Add("빠른 추가").SetAccelerator(quickAddShortcut).OnClick(func(*application.Context) { openQuick() })
	menu.Add("할 일 창 열기").SetAccelerator(showTasksShortcut).OnClick(func(*application.Context) { showCentered() })
	menu.Add("웹 보드 열기").OnClick(func(*application.Context) { _ = betty.OpenBoard() })
	menu.Add("새로고침").OnClick(func(*application.Context) { go betty.Refresh() })
	menu.AddSeparator()
	autostart := menu.AddCheckbox("로그인 시 자동 실행", false)
	if on, err := app.Autostart.IsEnabled(); err == nil {
		autostart.SetChecked(on)
	}
	autostart.OnClick(func(*application.Context) {
		var err error
		if autostart.Checked() {
			err = app.Autostart.Enable()
		} else {
			err = app.Autostart.Disable()
		}
		if err != nil {
			autostart.SetChecked(!autostart.Checked())
			app.Dialog.Error().SetTitle("자동 실행").SetMessage(err.Error()).Show()
		}
	})
	menu.Add("로그아웃").OnClick(func(*application.Context) { _ = betty.SignOut() })
	menu.AddSeparator()
	menu.Add("Betty 종료").SetAccelerator("CmdOrCtrl+Q").OnClick(func(*application.Context) { app.Quit() })
	tray.SetMenu(menu)

	if err := app.GlobalShortcut.Register(quickAddShortcut, openQuick); err != nil {
		log.Printf("quick add shortcut %s unavailable: %v", quickAddShortcut, err)
	}
	if err := app.GlobalShortcut.Register(showTasksShortcut, toggleCentered); err != nil {
		log.Printf("task window shortcut %s unavailable: %v", showTasksShortcut, err)
	}

	// Signed out: show the sign-in window right away instead of waiting for a menu bar click.
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		if !betty.View().SignedIn {
			showCentered()
		}
	})

	// Clicking a "new task" notification opens the task list.
	ns.OnNotificationResponse(func(notifications.NotificationResult) { tray.ShowWindow() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
