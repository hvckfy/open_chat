package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	appstate "openchat/internal/app"
	"openchat/pkg/client"
)

// runOnboarding shows whichever of {welcome, create+backup, import,
// set-PIN, unlock} screens is appropriate, and calls onReady with the
// recovered mnemonic once the user has an unlocked wallet. It owns
// win.SetContent for the whole flow.
func runOnboarding(win fyne.Window, store *appstate.WalletStore, onReady func(mnemonic string)) {
	exists, err := store.Exists()
	if err != nil {
		dialog.ShowError(fmt.Errorf("checking for existing wallet: %w", err), win)
		return
	}
	if exists {
		showUnlockScreen(win, store, onReady)
		return
	}
	showWelcomeScreen(win, store, onReady)
}

func showWelcomeScreen(win fyne.Window, store *appstate.WalletStore, onReady func(string)) {
	title := widget.NewLabelWithStyle("Welcome to OpenChat", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabel("A decentralized, end-to-end encrypted messenger.\nMessages are free to send — there's no token or fee.")
	subtitle.Alignment = fyne.TextAlignCenter
	subtitle.Wrapping = fyne.TextWrapWord

	createBtn := widget.NewButton("Create a new identity", func() {
		w, err := client.NewWallet("", "")
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		showBackupScreen(win, w.Keys.Mnemonic, func() {
			showSetPinScreen(win, store, w.Keys.Mnemonic, onReady)
		})
	})
	createBtn.Importance = widget.HighImportance

	importBtn := widget.NewButton("I already have a recovery phrase", func() {
		showImportScreen(win, store, onReady)
	})

	content := container.NewCenter(container.NewVBox(
		title, subtitle,
		container.NewGridWithColumns(1, createBtn, importBtn),
	))
	win.SetContent(content)
}

func showBackupScreen(win fyne.Window, mnemonic string, onContinue func()) {
	title := widget.NewLabelWithStyle("Save your recovery phrase", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	warning := widget.NewLabel("Write these 24 words down and keep them somewhere safe.\n" +
		"Anyone with this phrase can read your messages and impersonate you.\n" +
		"It is the ONLY way to recover your account — OpenChat cannot reset it for you.")
	warning.Wrapping = fyne.TextWrapWord
	warning.Alignment = fyne.TextAlignCenter

	phrase := widget.NewMultiLineEntry()
	phrase.SetText(mnemonic)
	phrase.Wrapping = fyne.TextWrapWord
	// Deliberately left enabled rather than phrase.Disable(): a disabled
	// Entry renders with theme.DisabledColor(), which on some OS
	// light/dark theme combinations ends up low-contrast-to-invisible —
	// exactly the wrong failure mode for the one screen showing the
	// recovery phrase. Reverting any edit keeps this effectively
	// read-only while guaranteeing normal, theme-correct, selectable text.
	makeReadOnly(phrase, mnemonic)

	confirmed := widget.NewCheck("I've written down my recovery phrase", nil)

	cont := widget.NewButton("Continue", func() {
		if !confirmed.Checked {
			dialog.ShowInformation("Not so fast", "Please confirm you've saved your recovery phrase first.", win)
			return
		}
		onContinue()
	})
	cont.Importance = widget.HighImportance

	content := container.NewBorder(
		container.NewVBox(title, warning), cont, nil, nil,
		container.NewVBox(phrase, confirmed),
	)
	win.SetContent(content)
}

func showImportScreen(win fyne.Window, store *appstate.WalletStore, onReady func(string)) {
	title := widget.NewLabelWithStyle("Import your recovery phrase", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	entry := widget.NewMultiLineEntry()
	entry.SetPlaceHolder("Enter your 12/24-word recovery phrase, separated by spaces")
	entry.Wrapping = fyne.TextWrapWord

	importBtn := widget.NewButton("Import", func() {
		mnemonic := entry.Text
		if _, err := client.NewWallet(mnemonic, ""); err != nil {
			dialog.ShowError(fmt.Errorf("that doesn't look like a valid recovery phrase: %w", err), win)
			return
		}
		showSetPinScreen(win, store, mnemonic, onReady)
	})
	importBtn.Importance = widget.HighImportance

	backBtn := widget.NewButton("Back", func() { showWelcomeScreen(win, store, onReady) })

	content := container.NewBorder(
		title, container.NewGridWithColumns(2, backBtn, importBtn), nil, nil,
		entry,
	)
	win.SetContent(content)
}

func showSetPinScreen(win fyne.Window, store *appstate.WalletStore, mnemonic string, onReady func(string)) {
	title := widget.NewLabelWithStyle("Set a PIN", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	info := widget.NewLabel("This PIN encrypts your recovery phrase on this device only.\nIt is not sent anywhere and cannot be recovered if forgotten\n(you'd re-import your recovery phrase instead).")
	info.Wrapping = fyne.TextWrapWord
	info.Alignment = fyne.TextAlignCenter

	pin1 := widget.NewPasswordEntry()
	pin1.SetPlaceHolder("PIN (min 4 digits)")
	pin2 := widget.NewPasswordEntry()
	pin2.SetPlaceHolder("Confirm PIN")

	confirmBtn := widget.NewButton("Save & Continue", func() {
		if len(pin1.Text) < 4 {
			dialog.ShowInformation("PIN too short", "Please use at least 4 characters.", win)
			return
		}
		if pin1.Text != pin2.Text {
			dialog.ShowInformation("PINs don't match", "Please re-enter matching PINs.", win)
			return
		}
		if err := store.Save(mnemonic, pin1.Text); err != nil {
			dialog.ShowError(err, win)
			return
		}
		onReady(mnemonic)
	})
	confirmBtn.Importance = widget.HighImportance

	content := container.NewCenter(container.NewVBox(title, info, pin1, pin2, confirmBtn))
	win.SetContent(content)
}

func showUnlockScreen(win fyne.Window, store *appstate.WalletStore, onReady func(string)) {
	title := widget.NewLabelWithStyle("Enter your PIN", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	pin := widget.NewPasswordEntry()
	pin.SetPlaceHolder("PIN")

	var unlockBtn *widget.Button
	doUnlock := func() {
		mnemonic, err := store.Load(pin.Text)
		if err != nil {
			dialog.ShowError(err, win)
			pin.SetText("")
			return
		}
		onReady(mnemonic)
	}
	pin.OnSubmitted = func(string) { doUnlock() }
	unlockBtn = widget.NewButton("Unlock", doUnlock)
	unlockBtn.Importance = widget.HighImportance

	content := container.NewCenter(container.NewVBox(title, pin, unlockBtn))
	win.SetContent(content)
}
