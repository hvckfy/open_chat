// Command app is the OpenChat GUI messenger: the same Go codebase builds
// natively for macOS, Windows, Linux, iOS and Android via Fyne (see
// README.md "Desktop & mobile app" for the exact `fyne package` /
// `fyne-cross` commands per platform). All chain/network/crypto logic is
// reused unmodified from pkg/client and pkg/crypto — this package is
// purely the UI plus small local-storage glue in internal/app.
package main

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	appstate "openchat/fyne/internal/app"
	"openchat/pkg/client"
	"openchat/pkg/tlsutil"
)

func main() {
	a := app.NewWithID("network.openchat.app")
	win := a.NewWindow("OpenChat")
	win.Resize(fyne.NewSize(900, 600))

	walletStore := appstate.NewWalletStore()

	// sessionCancel stops whatever session (background Listen loop) is
	// currently running. It's reassigned every time startSession begins a
	// new session (initial login, or a fresh one after Log out), and
	// invoked both when the window closes and when the user logs out —
	// each session gets its own context, since a canceled context can
	// never be "un-canceled" for the next login.
	var sessionCancel context.CancelFunc
	win.SetOnClosed(func() {
		if sessionCancel != nil {
			sessionCancel()
		}
	})

	var startSession func()
	startSession = func() {
		runOnboarding(win, walletStore, func(mnemonic string) {
			ctx, cancel := context.WithCancel(context.Background())
			sessionCancel = cancel
			startApp(ctx, a, win, mnemonic, walletStore, func() {
				cancel()
				startSession()
			})
		})
	}
	startSession()

	win.ShowAndRun()
}

// startApp builds the wallet, network client and local chat store from an
// unlocked mnemonic, then switches the window over to the main screen.
// onLogout is wired to the main screen's "Log out" action.
func startApp(ctx context.Context, a fyne.App, win fyne.Window, mnemonic string, walletStore *appstate.WalletStore, onLogout func()) {
	wallet, err := client.NewWallet(mnemonic, "")
	if err != nil {
		dialog.ShowError(fmt.Errorf("deriving wallet: %w", err), win)
		return
	}

	netSettings := loadNetworkSettings(a)
	tlsCreds, err := tlsutil.ClientTLS(netSettings.CAPath, netSettings.Insecure)
	if err != nil {
		dialog.ShowError(fmt.Errorf("TLS setup: %w", err), win)
		return
	}
	netClient := client.New(netSettings.Bootstrap, tlsCreds)

	store, err := appstate.NewChatStore()
	if err != nil {
		dialog.ShowError(fmt.Errorf("opening local chat history: %w", err), win)
		return
	}

	svc := appstate.NewService(wallet, netClient, store)

	// Wire the UI's callbacks (showMainScreen sets svc.OnMessage/OnStatus
	// synchronously) before starting the background Listen loop, so no
	// early event can race past a nil callback.
	showMainScreen(ctx, win, svc, func() {
		wipeSessionData(walletStore, store, win, onLogout)
	})
	svc.Start(ctx)
}

// wipeSessionData deletes the on-disk wallet and chat history and then
// hands off to onLogout (which stops the current session's background
// work and returns to onboarding). Both deletions are best-effort logged
// rather than fatal — a user who explicitly asked to log out should still
// end up back at the welcome screen even if, say, one file was already
// missing or briefly locked by the OS.
func wipeSessionData(walletStore *appstate.WalletStore, store *appstate.ChatStore, win fyne.Window, onLogout func()) {
	if err := walletStore.Delete(); err != nil {
		dialog.ShowError(fmt.Errorf("logout: removing wallet: %w", err), win)
	}
	if err := store.Wipe(); err != nil {
		dialog.ShowError(fmt.Errorf("logout: clearing chat history: %w", err), win)
	}
	onLogout()
}
