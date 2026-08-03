package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	appstate "openchat/internal/app"
)

// mainScreen is the messenger's primary view: a contact list on the left,
// the selected conversation on the right, and a small toolbar/status bar.
// All Service callbacks arrive on background goroutines and are marshaled
// back onto the Fyne main goroutine via fyne.Do before touching widgets.
type mainScreen struct {
	win      fyne.Window
	svc      *appstate.Service
	ctx      context.Context
	onLogout func()

	contacts     []*appstate.Contact
	contactList  *widget.List
	selected     *appstate.Contact
	messagesView *fyne.Container
	messagesScroll *container.Scroll
	input        *widget.Entry
	statusLabel  *widget.Label
}

// showMainScreen builds the messenger's primary view. onLogout is invoked
// once the user has confirmed they want to log out — it must wipe local
// session data and return to onboarding (see cmd/app/main.go).
func showMainScreen(ctx context.Context, win fyne.Window, svc *appstate.Service, onLogout func()) {
	m := &mainScreen{win: win, svc: svc, ctx: ctx, onLogout: onLogout}
	m.build()

	svc.OnMessage = func(from string) {
		fyne.Do(func() {
			m.refreshContacts()
			if m.selected != nil && m.selected.Address == from {
				m.refreshMessages()
			}
		})
	}
	svc.OnStatus = func(text string) {
		fyne.Do(func() { m.statusLabel.SetText(text) })
	}

	m.refreshContacts()
}

func (m *mainScreen) build() {
	m.statusLabel = widget.NewLabel("connecting…")
	m.statusLabel.Wrapping = fyne.TextWrapWord

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() { m.showAddContact() }),
		widget.NewToolbarAction(theme.AccountIcon(), func() { m.showMyCode() }),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			showNetworkSettingsDialog(m.win, func(networkSettings) {
				dialog.ShowInformation("Restart required", "Restart OpenChat for the new network settings to take effect.", m.win)
			})
		}),
		widget.NewToolbarAction(theme.LogoutIcon(), func() { m.showLogout() }),
	)

	m.contactList = widget.NewList(
		func() int { return len(m.contacts) },
		func() fyne.CanvasObject {
			return newContactRow()
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			c := m.contacts[i]
			row := o.(*contactRow)
			row.label.SetText(c.DisplayName)
			row.editBtn.OnTapped = func() { m.showEditContact(c) }
		},
	)
	m.contactList.OnSelected = func(i widget.ListItemID) {
		m.selected = m.contacts[i]
		m.refreshMessages()
	}

	left := container.NewBorder(widget.NewLabelWithStyle("Chats", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, m.contactList)

	m.messagesView = container.NewVBox()
	m.messagesScroll = container.NewVScroll(m.messagesView)

	m.input = widget.NewEntry()
	m.input.SetPlaceHolder("Type a message…")
	sendBtn := widget.NewButtonWithIcon("", theme.MailSendIcon(), m.send)
	m.input.OnSubmitted = func(string) { m.send() }

	right := container.NewBorder(nil, container.NewBorder(nil, nil, nil, sendBtn, m.input), nil, nil, m.messagesScroll)

	split := container.NewHSplit(left, right)
	split.Offset = 0.3

	content := container.NewBorder(toolbar, m.statusLabel, nil, nil, split)
	m.win.SetContent(content)
}

// contactRow is one row of m.contactList: a display-name label plus a
// small edit button. Embedding the fyne.CanvasObject built by
// container.NewBorder (rather than indexing into its Objects slice by
// position) means UpdateItem below reaches the label/button through
// named fields instead of a fragile, easy-to-get-wrong positional cast.
type contactRow struct {
	fyne.CanvasObject
	label   *widget.Label
	editBtn *widget.Button
}

func newContactRow() *contactRow {
	label := widget.NewLabel("contact")
	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), nil)
	editBtn.Importance = widget.LowImportance
	return &contactRow{
		CanvasObject: container.NewBorder(nil, nil, nil, editBtn, label),
		label:        label,
		editBtn:      editBtn,
	}
}

func (m *mainScreen) refreshContacts() {
	m.contacts = m.svc.Store.Contacts()
	m.contactList.Refresh()
}

func (m *mainScreen) refreshMessages() {
	m.messagesView.Objects = nil
	if m.selected == nil {
		m.messagesView.Refresh()
		return
	}
	for _, msg := range m.svc.Store.Messages(m.selected.Address) {
		m.messagesView.Add(renderMessageBubble(m.win, msg))
	}
	m.messagesView.Refresh()
	m.messagesScroll.ScrollToBottom()
}

func renderMessageBubble(win fyne.Window, msg *appstate.StoredMessage) fyne.CanvasObject {
	who := "Them"
	align := fyne.TextAlignLeading
	if msg.Direction == appstate.Outgoing {
		who = "You"
		align = fyne.TextAlignTrailing
	}
	ts := time.UnixMilli(msg.Timestamp).Local().Format("15:04")
	label := widget.NewLabel(fmt.Sprintf("%s · %s\n%s", who, ts, msg.Text))
	label.Wrapping = fyne.TextWrapWord
	label.Alignment = align

	text := msg.Text
	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		win.Clipboard().SetContent(text)
	})
	copyBtn.Importance = widget.LowImportance

	return container.NewBorder(nil, nil, nil, copyBtn, label)
}

func (m *mainScreen) send() {
	text := m.input.Text
	if text == "" || m.selected == nil {
		return
	}
	m.input.SetText("")
	target := m.selected
	go func() {
		err := m.svc.SendText(m.ctx, target, text)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, m.win)
				return
			}
			if m.selected != nil && m.selected.Address == target.Address {
				m.refreshMessages()
			}
		})
	}()
}

func (m *mainScreen) showAddContact() {
	codeEntry := widget.NewMultiLineEntry()
	codeEntry.SetPlaceHolder("Paste their OpenChat contact code (oc1:...)")
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Name (optional)")

	form := widget.NewForm(
		widget.NewFormItem("Contact code", codeEntry),
		widget.NewFormItem("Name", nameEntry),
	)

	d := dialog.NewCustomConfirm("Add contact", "Add", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		parsed, err := appstate.ParseContactCode(codeEntry.Text)
		if err != nil {
			dialog.ShowError(err, m.win)
			return
		}
		name := nameEntry.Text
		if name == "" {
			name = parsed.Address[:12]
		}
		c := &appstate.Contact{Address: parsed.Address, X25519PubHex: parsed.X25519PubHex, DisplayName: name}
		if err := m.svc.Store.UpsertContact(c); err != nil {
			dialog.ShowError(err, m.win)
			return
		}
		m.refreshContacts()
	}, m.win)
	// Without an explicit size, ShowCustomConfirm/NewCustomConfirm sizes
	// the dialog to the form's natural MinSize, which for a two-row form
	// can shrink to a tiny square on some platforms/themes. Every other
	// custom dialog in this app (showMyCode, showNetworkSettingsDialog)
	// sets an explicit size for exactly this reason.
	d.Resize(fyne.NewSize(480, 280))
	d.Show()
}

// showEditContact lets the user rename an existing contact — most often
// used on the placeholder name ("a1b2c3…d4e5f6") ChatStore.AppendMessage
// gives a contact the first time it hears from an address it didn't
// already know, before the user has had a chance to add a proper name.
// The address/encryption key themselves aren't editable here: they're
// the contact's actual cryptographic identity, not a label.
func (m *mainScreen) showEditContact(c *appstate.Contact) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(c.DisplayName)

	addr := widget.NewLabel(c.Address)
	addr.Wrapping = fyne.TextWrapBreak

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Address", addr),
	)

	d := dialog.NewCustomConfirm("Edit contact", "Save", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			name = c.Address[:12]
		}
		updated := &appstate.Contact{Address: c.Address, X25519PubHex: c.X25519PubHex, DisplayName: name}
		if err := m.svc.Store.UpsertContact(updated); err != nil {
			dialog.ShowError(err, m.win)
			return
		}
		m.refreshContacts()
	}, m.win)
	// Same reasoning as showAddContact below: an unsized custom-content
	// dialog can shrink to a tiny square on some platforms/themes.
	d.Resize(fyne.NewSize(480, 240))
	d.Show()
}

// showLogout confirms, then wipes this device's wallet and chat history
// and returns to onboarding — "Log out" in the toolbar.
func (m *mainScreen) showLogout() {
	dialog.ShowConfirm(
		"Log out",
		"This erases your wallet and message history from this device.\n"+
			"Make sure you've saved your recovery phrase — OpenChat cannot recover it for you.\n\n"+
			"Continue?",
		func(ok bool) {
			if !ok || m.onLogout == nil {
				return
			}
			m.onLogout()
		},
		m.win,
	)
}

func (m *mainScreen) showMyCode() {
	code := m.svc.MyContactCode()
	entry := widget.NewMultiLineEntry()
	entry.SetText(code)
	entry.Wrapping = fyne.TextWrapWord
	// Deliberately left enabled rather than entry.Disable(): a disabled
	// Entry renders with theme.DisabledColor(), which on some OS
	// light/dark theme combinations ends up low-contrast-to-invisible
	// against the dialog background. Reverting any edit keeps this
	// effectively read-only while guaranteeing normal, theme-correct
	// (and selectable/copyable) text.
	makeReadOnly(entry, code)

	copyBtn := widget.NewButton("Copy to clipboard", func() {
		m.win.Clipboard().SetContent(code)
	})

	content := container.NewVBox(
		widget.NewLabel("Share this code so others can message you:"),
		entry,
		copyBtn,
		widget.NewLabel("Your address: "+m.svc.MyAddress()),
	)
	d := dialog.NewCustom("My contact code", "Close", content, m.win)
	d.Resize(fyne.NewSize(420, 260))
	d.Show()
}

// makeReadOnly keeps entry enabled (so it always renders with the theme's
// normal foreground color, unlike a Disable()'d Entry — see the callers
// above/in onboarding.go for why that matters) while preventing the user
// from actually changing its content: any edit is immediately reverted
// back to want.
func makeReadOnly(entry *widget.Entry, want string) {
	entry.OnChanged = func(cur string) {
		if cur != want {
			entry.SetText(want)
		}
	}
}
