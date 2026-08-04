# OpenChat — design brief (functional spec, no visuals)

This document describes **what exists on each screen and dialog and what
it needs to do** — not how it should look. It's written to be handed to a
design tool (claude-design) to produce the actual visual design system and
templates; once those come back, the native macOS/iOS build (`app/xcode`)
starts from them.

It describes the *product's* current functionality (implemented today in
the reference Fyne app, `app/fyne/cmd/app`), not that specific
implementation's widget choices — a native redesign is free to lay
anything out however fits each platform's conventions, as long as every
function listed below has a place to live.

## Product summary

OpenChat is a decentralized, end-to-end encrypted messenger. There is no
central server, no account/phone-number/email signup, no fee to send a
message, and no company that can read messages or reset a lost account.
Identity **is** a cryptographic keypair, derived from a 24-word recovery
phrase the user generates and backs up themselves. Two things follow from
this that the design needs to take seriously:

- **The recovery phrase is the whole account.** Losing it means losing
  the account, permanently, with no recovery path. The backup step is the
  single highest-stakes moment in the whole app and should read as such —
  not buried in a generic "next" flow.
- **There's no username directory.** You message someone by having their
  contact code (their address + encryption key, packaged as one shareable
  string). "Add contact" is closer to "add a phone number" than "search
  for a user."

## Platforms & constraints

- **Today:** one Go codebase (Fyne) builds the same UI natively for
  macOS, Windows, Linux, iOS, and Android — functional, but Fyne's widget
  set doesn't attempt to look platform-native anywhere.
- **Planned:** platform-native shells (Xcode/SwiftUI for macOS+iOS first)
  that talk to the same node network and reuse the same protocol/crypto
  logic, with real per-platform design. This brief is written for that —
  assume normal native idioms (navigation split view on macOS/iPad, stack
  navigation on iPhone, sheets/alerts, system dark/light mode, Dynamic
  Type / accessibility) rather than copying Fyne's layout literally.
- **Out of scope for this pass** (not implemented, don't design for them
  unless separately requested): group chats, media/file attachments, read
  receipts/typing indicators, push notifications, message editing or
  deletion, search.

## Global structure

Two top-level states, mutually exclusive, filling the whole window/screen
(no persistent chrome like a tab bar spans both):

1. **Onboarding** — shown when there's no unlocked wallet yet (first
   launch, or right after logging out). A linear flow, not a
   freely-navigable set of tabs.
2. **Main app** — shown once a wallet is unlocked. A messenger: contact
   list + conversation, plus account-level actions (add contact, share my
   code, settings, log out).

There is no splash/loading screen on launch — the app decides
synchronously (a local file either exists or doesn't) which of the two
states to show.

---

## A. Onboarding flow

Linear, one screen fully replaces the previous — think wizard, not tabs.
Returning users (a wallet already exists on this device) skip straight to
**A5 Unlock**; everyone else starts at **A1 Welcome**.

### A1 — Welcome

**Purpose:** entry point; branding + the two ways to get an identity.

**Contains:**
- Product name/wordmark.
- One line explaining what OpenChat is (decentralized, E2EE, free to
  send — no fee/token).
- Primary action: **"Create a new identity"**.
- Secondary action: **"I already have a recovery phrase"**.

**Behavior:** "Create" generates a brand-new keypair and mnemonic
immediately (no form to fill first) and proceeds to A2. "I already have a
recovery phrase" proceeds to A3.

### A2 — Back up your recovery phrase

**Purpose:** the account's only backup, shown exactly once at creation.
This is the screen to treat with the most visual weight in the entire
flow.

**Contains:**
- Explicit warning copy: write these words down, anyone with them can
  read your messages and impersonate you, this is the *only* recovery
  method, OpenChat cannot reset it.
- The 24-word recovery phrase itself, displayed in full, selectable/
  copyable — **not** styled as a disabled/greyed-out field (a real bug
  we hit: disabled-looking text can render illegibly against some
  dark/light theme combinations; the phrase must always render at full,
  normal-text contrast even though it isn't editable).
- A confirmation checkbox: "I've written down my recovery phrase."
- Primary action: **Continue** — disabled/blocked until the checkbox is
  checked.

**Behavior:** proceeds to A4 (set PIN) on continue. No skip option.

### A3 — Import your recovery phrase

**Purpose:** restore an existing identity on a new device, or after a
logout.

**Contains:**
- Multi-line text input for the recovery phrase (12 or 24 words).
- Primary action: **Import**.
- Secondary action: **Back** (returns to A1).

**Behavior:** validates the phrase on submit; an invalid phrase shows an
inline/dialog error and keeps the user on this screen (specific reason if
knowable, e.g. wrong word count or a bad checksum word). Valid phrase
proceeds to A4.

### A4 — Set a PIN

**Purpose:** a local-only PIN that encrypts the recovery phrase at rest
on *this device*. It's not an account password — it can't be reset or
recovered by anyone, and forgetting it just means re-importing the
recovery phrase from A3.

**Contains:**
- Explanatory copy making the "local-only, not recoverable, not sent
  anywhere" point explicit.
- PIN entry (masked).
- Confirm-PIN entry (masked).
- Primary action: **Save & Continue**.

**Input:** the on-screen numeric keypad is the primary, always-visible
affordance (works identically on every platform, including with no
physical keyboard — a touchscreen or a Mac with an on-screen-only
input). Where a physical/external keyboard is present (Mac, iPad with a
keyboard attached), its number keys and Delete/Backspace also drive the
same PIN entry, as a convenience — not a replacement for the on-screen
keypad.

**Behavior:** validates minimum length (4+ characters) and that both
entries match, with inline errors for each failure. On success, encrypts
and persists the wallet locally and proceeds straight into the main app
(B1) — no separate "success" screen.

### A5 — Unlock

**Purpose:** re-entry for a device that already has a saved wallet
(normal app launch after the first one).

**Contains:**
- PIN entry (masked), auto-focused.
- Primary action: **Unlock**.

**Behavior:** submitting an empty/wrong PIN shows an error and clears the
field; a correct PIN proceeds straight to the main app (B1). There is
intentionally no "forgot PIN" link here — recovery is "log out" (destroys
local data) then A3 import, and that path should probably be discoverable
somewhere on this screen even though it's destructive, so a genuinely
locked-out user isn't stuck.

---

## B. Main app

### B1 — Chat list + conversation

**Purpose:** the primary, default screen once unlocked. Two-pane on
larger screens (contact list + active conversation side by side); on a
phone-sized screen this should become push/pop navigation (list → tap a
contact → conversation, with a back action), not a permanent split.

**Contains — contact list (left/primary pane):**
- One row per saved contact: display name (falls back to a shortened
  address if the contact has never been named — see B4). Rows are
  tappable to open that conversation. The inline "edit" affordance is
  platform-native rather than one shared control: reveal-on-hover pencil
  on macOS/iPad, reveal-on-swipe on iPhone (native implementations: a
  permanently-visible button on every row read as clutter and, on
  iPhone, competed with the row's own tap-to-open gesture and disclosure
  chevron).
- Empty state: no contacts yet — should point at "Add contact" (B3)
  rather than just being a blank pane.
- A persistent, small connection/status indicator somewhere in this
  screen's chrome (e.g. "connecting…", "message history up to date", or
  an error) — this is the only place a global network status shows.

**Contains — conversation (right/secondary pane):**
- Header identifying who this conversation is with (name, plus avatar on
  iPhone where there's no persistent sidebar row to look at) — on
  iPhone this is also where the per-contact "edit" affordance lives,
  since the swipe gesture on the list row is one screen away once a
  conversation is open.
- Message history for the selected contact, in chronological order,
  visually distinguished by direction (sent vs. received) — timestamp
  per message.
- Each message has a copy-to-clipboard affordance.
- Empty state: contact selected but no messages yet.
- No-selection state (larger screens only): nothing selected yet.
- Composer: auto-growing text input (starts at one line, expands upward
  as the typed text needs more room, up to a cap before it scrolls
  internally — no visible scrollbar) + send action at the bottom.
  Return/Enter submits on desktop-class input (physical keyboard);
  Shift+Return inserts a line break instead. The explicit send button
  always works regardless of input method.

**Contains — toolbar/header actions (available from this screen,
independent of which contact is selected):**
- **Add contact** → opens B3.
- **My code** (share how others reach me) → opens B5.
- **Settings** (network settings) → opens B6.
- **Log out** → opens B7.

**Behavior:** sending a message clears the composer immediately (don't
wait for network confirmation to clear input) and appends it to the
conversation optimistically; a send failure surfaces as an error without
silently losing the typed text's *intent* (the user should be able to
tell it didn't go through). Incoming messages update the contact list
(most-recently-active ordering is reasonable, though not currently
implemented) and, if that conversation is currently open, append live.
On launch, the app silently recovers any message history missed while
closed before the live connection takes over — this can be represented
as a brief "syncing…" status rather than a blocking screen.

### B2 — Message bubble (component, not a standalone screen)

**Purpose:** the atomic unit of the conversation view.

**Contains:** message text, sender (implicit via direction/alignment),
timestamp, copy action. No read receipts, no edit/delete, no reactions —
none of that exists in the underlying protocol today.

### B3 — Add contact (dialog)

**Purpose:** add someone by their contact code (address + encryption key,
one shareable string) rather than searching a directory that doesn't
exist.

**Contains:**
- Text input for pasting a contact code.
- Optional name field (falls back to a shortened address if left blank).
- Confirm / Cancel actions.

**Behavior:** an invalid/malformed code shows an inline error and doesn't
close the dialog. A valid code adds the contact and returns to B1 with
that contact now in the list.

### B4 — Edit contact (dialog)

**Purpose:** rename a contact — most commonly used the first time
someone messages you unprompted and shows up with a placeholder name
(shortened address) instead of a real one.

**Contains:**
- Editable name field.
- Read-only display of the contact's address (not editable — it's their
  cryptographic identity, not a label).
- Save / Cancel actions.

### B5 — My contact code (dialog)

**Purpose:** the "how do people reach me" screen — the counterpart to B3
on the other side of an introduction.

**Contains:**
- Explanatory line ("share this so others can message you").
- The contact code itself, displayed in full and copyable (same
  full-contrast/never-disabled-looking treatment as A2's recovery
  phrase — it's not secret, but it is the thing being read/copied).
- Explicit "copy to clipboard" action.
- The user's raw address shown separately/secondarily, for cases where
  only the address (not the full code) is needed.
- Close action.

### B6 — Network settings (dialog)

**Purpose:** point the app at a specific set of gateway nodes instead of
the built-in defaults — mainly a power-user / self-hosted-network
feature, not a first-run requirement.

**Contains:**
- Bootstrap gateway list (editable, comma-separated or equivalent
  structured input).
- Optional CA certificate, for a private/self-signed network — a file
  picker everywhere; platforms with a real filesystem/typed-path
  convention (macOS) additionally allow typing the path directly, but
  platforms without one (iOS: files only ever come from the picker) show
  just the picked filename, not an editable path field with nothing
  meaningful to type into it.
- "Skip TLS verification" toggle, explicitly labeled as a local/dev-only
  affordance (should look like an advanced/danger option, not a default
  peer of the other fields).
- Save / Cancel actions.

**Behavior:** saving requires an app restart to take effect (surfaced as
an explicit message, not silent) — the current network session doesn't
hot-swap.

### B7 — Log out (confirmation)

**Purpose:** destructively wipes this device's wallet and message
history and returns to onboarding (A1). This is a "wipe this device,"
not "sign out of an account somewhere" — there is no server-side account
to sign out of.

**Contains:**
- Confirmation copy stating plainly what's about to be destroyed, and
  reminding the user their recovery phrase is the only way back in.
- Confirm / Cancel actions.

**Behavior:** on confirm, deletes the local wallet and chat history and
returns straight to A1 — no intermediate "are you really sure" beyond
this one confirmation.

---

## Cross-cutting notes for the design system

- **No paywalls, no pricing, no "free tier" language anywhere** — every
  message is free by design; don't design any affordance that implies
  otherwise.
- **Security-sensitive text must never look disabled/greyed.** Any
  read-only-but-important text (recovery phrase, contact code) needs a
  treatment that's visually read-only (not editable-looking) without
  using the platform's generic "disabled" appearance, which can render
  at illegible contrast in some theme combinations.
- **Errors are per-action, not global.** Every destructive or fallible
  action (import phrase, add contact, unlock, send) needs its own
  specific error surface near the point of failure, not a single global
  error banner.
- **Support system light/dark mode.** Nothing in the product implies a
  fixed theme.
- **Address-as-identity is a recurring visual element.** Addresses are
  long hex strings; the design should establish one consistent
  abbreviated-address treatment (e.g. truncation pattern) used everywhere
  an unnamed contact or a raw address appears (B1 empty contact names, B4
  read-only address, B5 "your address").

## Reference iconography (current implementation, non-binding)

Add contact, share/my-code, settings (gear), log out, send, edit
(pencil/document), copy — a native redesign should use each platform's
own icon conventions for these same seven actions rather than matching
these literally.

## Data model quick reference

For understanding what's static copy vs. dynamic content when designing:

- **Contact:** address (hex), encryption key (hex, may be empty until
  the contact has messaged at least once), display name.
- **Message:** direction (sent/received), text, timestamp. No delivery/
  read status beyond "it's in the local history," no attachments.
