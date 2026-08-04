# OpenChat — UX/UI design specification

Companion to `design-code.md` (functional spec). This document adds the
concrete visual/interaction layer — layout numbers, spacing, typography,
color tokens, component states — needed to build native shells,
starting with macOS/iOS (SwiftUI), with Windows/Linux/Android to follow
the same tokens under their own native controls.

Brand tone: plain, warm, everyday messenger. The blockchain/crypto layer
is real but invisible in copy and visuals — no ledger/chain iconography,
no jargon beyond what the functional spec already requires (recovery
phrase, contact code).

---

## 1. Design tokens

### 1.1 Color

Defined in OKLCH, warm-neutral base (slight warm tint, chroma ≤ 0.02),
one accent hue (indigo, ~265°) used at consistent chroma/lightness across
light/dark. Implement as semantic tokens, not literal colors, so each
native shell maps them to its own system (Assets.xcassets color sets on
Apple, resource dictionaries on Windows, GTK theme on Linux).

| Token | Light | Dark | Usage |
|---|---|---|---|
| `bg.canvas` | oklch(0.98 0.005 80) | oklch(0.16 0.006 80) | window/screen background |
| `bg.surface` | oklch(1.0 0.003 80) | oklch(0.21 0.006 80) | cards, message bubbles (received), list rows |
| `bg.surface2` | oklch(0.955 0.006 80) | oklch(0.26 0.007 80) | sidebar, grouped-list background, composer bar |
| `bg.overlay` | oklch(0 0 0 / 0.32) | oklch(0 0 0 / 0.55) | sheet/dialog scrim |
| `border.hairline` | oklch(0.88 0.006 80) | oklch(0.32 0.008 80) | 1px separators |
| `text.primary` | oklch(0.22 0.006 80) | oklch(0.95 0.006 80) | body/title text |
| `text.secondary` | oklch(0.48 0.006 80) | oklch(0.68 0.006 80) | timestamps, captions, placeholders |
| `text.tertiary` | oklch(0.62 0.006 80) | oklch(0.54 0.006 80) | disabled labels, hint text |
| `accent` | oklch(0.55 0.15 265) | oklch(0.72 0.15 265) | primary actions, sent bubbles, links, focus ring |
| `accent.pressed` | oklch(0.47 0.15 265) | oklch(0.64 0.15 265) | pressed/active state of accent controls |
| `on-accent` | oklch(0.99 0.005 80) | oklch(0.14 0.02 265) | text/icon on top of `accent` |
| `danger` | oklch(0.55 0.17 25) | oklch(0.7 0.16 25) | destructive actions, errors |
| `success` | oklch(0.58 0.13 150) | oklch(0.7 0.13 150) | "connected / up to date" status |
| `warning` | oklch(0.75 0.14 80) | oklch(0.78 0.13 80) | "connecting…", advanced/danger toggles |

Read-only-but-important text (recovery phrase, contact code, addresses)
uses `text.primary` on `bg.surface2` in a monospaced style — never
`text.tertiary` or a platform "disabled" text color, per the functional
spec's contrast requirement.

### 1.2 Typography

Native shells use the system font at system sizes (SF Pro on Apple
platforms via Dynamic Type text styles; Segoe UI Variable on Windows;
Roboto/system on Android; the Linux DE's default on Linux). Numbers below
are the Apple/SwiftUI mapping; other platforms substitute their own scale
at equivalent roles.

| Role | SwiftUI text style | Size (pt) | Weight | Used for |
|---|---|---|---|---|
| Display | `.largeTitle` | 34 | Bold | A1 wordmark, A2 screen title |
| Title | `.title2` | 22 | Semibold | Screen/dialog titles (A3–A5, B3–B7) |
| Headline | `.headline` | 17 | Semibold | Contact name in list row, message sender emphasis |
| Body | `.body` | 17 | Regular | Message text, input fields, body copy |
| Subheadline | `.subheadline` | 15 | Regular | Secondary row text, dialog explanatory copy |
| Caption | `.caption` | 12 | Regular | Timestamps, status indicator, field hints |
| Mono | `.body` w/ `.monospaced` design | 15–17 | Regular | Recovery phrase, contact code, addresses, hex keys |

All text styles must scale with Dynamic Type (support at least up to
`.accessibility1`); layouts using fixed-height rows must switch to
intrinsic/auto height above the default type size rather than clipping.

### 1.3 Spacing & geometry

8pt base grid (4pt for tight component-internal spacing).

- Screen/dialog outer margin: 24pt (macOS/iPad), 20pt (iPhone).
- Stack spacing between grouped elements: 16pt; within a tight group
  (label + field): 8pt.
- Labeled single-line fields (Name, CA certificate path, etc.): the
  Caption label sits directly above the field, left-aligned, with no
  bordered container wrapping label + field together — just the label
  text, a small gap, then the field's own `bg.surface2` fill. The field
  itself has no fixed control-height frame forced on it and uses each
  platform's plain/borderless text-field style (AppKit's own bezeled
  border must be suppressed on macOS), so the field's box hugs the text
  closely instead of adding extra vertical padding that pushes the
  caret out of alignment with the visible text.
- Standard control height: 44pt (buttons, list rows, text fields) — also
  the minimum hit target on every platform.
- Corner radius: 10pt (buttons, text fields, small cards), 14pt (message
  bubbles, sheets/dialog panels), 20pt (large emphasis cards, e.g. A2's
  phrase card).
- Hairline separators: 1px (physical), inset 16pt from leading edge in
  lists (avatar width + gap), full-bleed elsewhere.
- Avatar/contact glyph: 40×40pt circle in list rows, 64×64pt in dialogs
  (My code / Edit contact), 96×96pt center-stage (none currently, reserved).

---

## 2. Platform shell & navigation

### 2.1 macOS

- Plain `HSplitView` (not `NavigationSplitView`), two columns (no third
  inspector column): **sidebar** (contact list) ideal width 280pt, min
  220pt, max 360pt (user-resizable, draggable divider — `HSplitView`'s
  native behavior, driven by each column's own `.frame(minWidth:idealWidth:maxWidth:)`);
  **detail** (conversation) fills the remainder, min 480pt.
- **The sidebar cannot be collapsed on macOS** — no toggle button, no
  keyboard shortcut; it's permanently visible. This is a deliberate
  departure from the platform's usual collapsible-sidebar convention,
  reached after `NavigationSplitView` (which is what macOS would
  normally use here) proved impossible to stop injecting its own default
  sidebar-toggle toolbar button, across three separate attempts: a
  custom "open-sided chevron" meant to replace the system glyph kept
  showing up duplicated alongside it regardless of how
  `.toolbar(removing: .sidebarToggle)` was attached; the plain system
  toggle left alone (with `columnVisibility` still bound) made the
  sidebar render as a separate floating/overlaid panel — visible desktop
  wallpaper in the gap around it — instead of a normal in-window
  collapse; and even removing the binding entirely while still calling
  `.toolbar(removing: .sidebarToggle)` left the system button showing up
  regardless. Switching macOS to plain `HSplitView` sidesteps the whole
  mechanism — it has no "sidebar" concept and no associated default
  toolbar item, so there's nothing left for the system to inject. iPad
  keeps `NavigationSplitView` with a normal collapsible sidebar and the
  system's own toggle, since it hasn't shown either problem.
- Toolbar (trailing, in the detail column's titlebar area since actions
  are global, not per-contact): Add contact, My code, Settings icons;
  Log out lives in an overflow/menu (⋯) or the app menu, not a persistent
  toolbar icon, since it's destructive and infrequent.
- Dialogs (B3–B7) are `.sheet` attached to the window, width 420–480pt,
  height intrinsic, presented from the detail column.
- Full system light/dark mode + accent color follows `accent` token
  (can also honor the user's macOS system accent color as an alternate
  mode — optional, not required for v1).

### 2.2 iOS / iPadOS

- **iPhone:** `NavigationStack`. B1's contact list is the stack root;
  tapping a row pushes the conversation (native push transition, back
  button reads the contact's name). Toolbar: leading — nothing (root);
  trailing — "+" (Add contact) and profile/code glyph (My code);
  Settings + Log out live under an overflow (⋯) menu or a Settings-style
  row surfaced from a leading avatar/menu button, to keep the root
  toolbar to 2 icons max.
- **iPad:** `NavigationSplitView` identical in structure to macOS
  (sidebar 320pt ideal, collapsible via the standard iPad split-view
  behavior); same toolbar placement as macOS.
- Dialogs (B3–B7) are `.sheet` with a single `.presentationDetents([.large])`
  on iPhone — always opens fully expanded. An earlier `[.medium, .large]`
  version (short dialogs like B4 Edit contact opening at `.medium`) was
  tried and dropped: it meant those sheets opened only half-expanded,
  which read as a bug more than an intentional compact state. Full
  `.sheet` (non-detent) on iPad, width 480pt.
- Onboarding (A1–A5) is presented full-screen, its own `NavigationStack`,
  no tab bar, no dismiss gesture on A2 (must complete the checkbox flow).
- Safe-area respecting; minimum 44×44pt tap targets throughout; keyboard
  avoidance on all text-entry screens (A3, A4, A5, B1 composer, B3, B4,
  B6).

### 2.3 Windows / Linux / Android (later pass, same tokens)

- Windows: NavigationView with a `Pane` (sidebar) using Fluent
  Mica/Acrylic surfaces mapped to `bg.surface2`; standard title bar.
- Linux: platform-native toolkit (GTK4/libadwaita `AdwNavigationSplitView`
  or Qt equivalent) with the same two-pane/sidebar structure.
- Android: `NavigationRail`/two-pane on large screens, single-pane +
  Material back stack on phones, mirroring the iOS phone behavior.
- All three reuse the color/spacing/type *roles* in this doc, expressed
  through each platform's own component library rather than Apple
  controls.

---

## 3. Screen-by-screen UX/UI detail

Sizes given as iPhone (390×844pt reference, Dynamic Type @ default) /
macOS (in the detail or full-window context). Same content, platform
idioms differ per §2.

### A1 — Welcome
- Centered vertical stack, safe-area top offset ~120pt (iPhone) / vertical
  center (macOS, in a fixed 480pt-wide column).
- Wordmark: Display style, `accent`-tinted glyph + `text.primary`
  wordmark, 16pt gap to tagline.
- Tagline: Subheadline, `text.secondary`, max width 320pt, center-aligned,
  2 lines max.
- Primary button "Create a new identity": full-width (iPhone, 20pt
  margins) / 280pt fixed (macOS), height 50pt, `accent` fill, `on-accent`
  text, radius 14pt (prominent — this is the one button in the whole app
  that should read as an emphasized "start" action, slightly taller than
  the standard 44pt control).
- Secondary "I already have a recovery phrase": same width, plain/
  bordered style, `text.primary` on `bg.surface`, 12pt gap below primary.
- No back/skip affordance (first screen).

### A2 — Back up your recovery phrase
- Highest visual weight screen in onboarding: `danger`-tinted warning
  block at top (icon + copy, `bg.surface2` panel, radius 14pt, 16pt
  padding), not a subtle caption — this is the one place a saturated
  warning color is justified.
- Phrase card: `bg.surface2`, radius 20pt, 20pt padding, 1px
  `border.hairline`. 24 words in a 3-column × 8-row grid (iPhone: 2×12),
  each word prefixed with its index (`1.`, `2.`…) in `text.secondary`
  Caption, the word itself in Mono/Body at `text.primary` — full contrast,
  selectable, long-press-to-copy per word or a single "Copy phrase"
  action for the whole block.
- Checkbox row directly under the card, 16pt gap: 22×22pt checkbox +
  "I've written down my recovery phrase." (Body, `text.primary`).
- Primary "Continue": disabled state = `accent` at 35% opacity + no
  pointer/tap feedback, not a desaturated gray (keep the button
  identifiable, just inert) until checkbox is checked.

### A3 — Import your recovery phrase
- Title (Title style) + one-line Subheadline explaining 12/24-word input.
- Multi-line text field: `bg.surface2`, radius 10pt, min height 120pt,
  Mono/Body text, placeholder "Enter your 12 or 24-word phrase…"
  (`text.tertiary`).
- Inline error (on invalid submit): `danger` Caption text directly under
  the field, field gets a 1.5pt `danger` border; specific message when
  known ("Wrong number of words", "Checksum word doesn't match").
- Primary "Import" (same button spec as A1 primary) + "Back" as a plain
  text button (Body, `accent`) top-leading or trailing per platform nav
  convention (nav-bar back item on iOS, plain button on macOS).

### A4 — Set a PIN
- Explanatory copy (Subheadline, `text.secondary`, 2 lines) stating
  local-only/non-recoverable, above the fields.
- PIN is numeric-only, 4+ digits (assumed from the functional spec's
  "4+ characters" — clarify with product if alphanumeric PINs are
  intended). Entry uses a **custom round numeric keypad**, never the
  system alphanumeric keyboard: a dot-mask progress row ("PIN" then
  "Confirm PIN", reusing the same keypad for both in sequence) above a
  3×4 grid of circular keys (1–9, blank, 0, delete), 60pt diameter on
  iPhone / 52pt on macOS, `bg.surface2` fill with 1px `border.hairline`,
  12–14pt gap, Body-weight digit + optional Caption letters (2:ABC…)
  for muscle-memory familiarity. This keypad is a shared component reused
  identically on A4 and A5 (and anywhere else PIN entry occurs) — same
  geometry, same key order, on every platform, including macOS/Windows/
  Linux where it replaces relying on the physical keyboard, so PIN entry
  reads identically everywhere.
- Dot-mask progress row: 16pt dots (not the smaller 9pt originally
  shipped — too small to read at a glance), horizontally centered above
  the keypad, with the "PIN"/"Confirm PIN" label centered above the dot
  row in turn — not left-aligned.
- **Input:** the on-screen keypad is the primary, always-visible input
  method (and the only one on iOS/iPadOS). On platforms with a physical
  keyboard (macOS, and eventually Windows/Linux), typing digits 0–9 and
  Delete/Backspace on the hardware keyboard is an additional, equivalent
  path to the same on-screen keys — a convenience for desktop users, not
  a replacement for the tappable keypad.
- Inline errors per field (too short / mismatch) in `danger` Caption
  directly under the offending field, not a single combined error.
- Primary "Save & Continue", same spec as A1 primary, full-width.

### A5 — Unlock
- Vertical center, tighter than A1 (returning-user screen, lower
  ceremony): small wordmark/glyph (24pt) above a masked PIN dot row
  (same 16pt centered dots as A4), then the same round numeric keypad
  component as A4, also accepting the physical keyboard on platforms
  that have one (see A4's Input note).
- No separate "Unlock" button — the keypad auto-submits once the PIN's
  expected length is reached (standard native passcode-entry pattern).
- Error state: wrong PIN clears the dots, shakes the row briefly (±6pt,
  120ms), shows Caption `danger` text below ("Incorrect PIN").
- "Log out and use a different recovery phrase" as a small plain-text
  link (Caption, `text.secondary`) bottom of screen/safe-area — present
  but visually de-emphasized versus the keypad.

### B1 — Chat list + conversation
**Sidebar / list column**
- Status bar at the very top of the sidebar, 32pt height, `bg.surface2`,
  Caption text + 8pt colored dot (`success`/`warning`/`danger`) — "Message
  history up to date" / "Connecting…" / "Connection error".
- List rows: 64pt height, 16pt leading inset. 40pt avatar (circular,
  initial-letter placeholder tinted by a deterministic hue from the
  address) + 12pt gap + name column (Headline `text.primary`, truncating
  middle for fallback-address names e.g. `0x4F2A…9E1C`) — no secondary
  line (no last-message preview in scope, since there's no metadata for
  it beyond what's open). Trailing: small edit (pencil) affordance,
  44×44pt tap target. On macOS it's a low-opacity button that reaches
  full opacity on row hover; on iOS there's no permanent on-row button
  at all — the row is a `NavigationLink` push, and edit lives behind a
  standard leading-to-trailing swipe action (`.swipeActions`), not a
  long-press.
- Empty state: centered, icon + "No contacts yet" (Headline) + "Add
  someone to start chatting" (Subheadline, `text.secondary`) + a
  secondary-style "Add contact" button.

**Conversation column**
- Header: contact name (Headline). On macOS, plain text leading in the
  toolbar/window title. On iOS, the nav bar's principal item carries a
  compact 28pt `ContactAvatar` + name together (so it's unambiguous who
  you're talking to, since the list is a full push-away, not a
  persistent sidebar), and the edit (pencil) affordance that lives on
  the list row on other platforms moves here instead, as a trailing
  nav-bar button — iPhone has nowhere else to put it once the row itself
  is gone behind the push.
- Message list: bottom-anchored, inverted scroll (newest at bottom,
  scroll position preserved/pinned to bottom on new message unless user
  has scrolled up). 16pt outer padding, 4pt gap between consecutive
  same-direction bubbles, 12pt gap when direction changes.
- Bubbles (see §B2 below) max width 72% of column width (macOS) / 78%
  (iPhone).
- No-selection state (macOS/iPad only, nothing picked): centered glyph +
  "Select a conversation" (Subheadline, `text.secondary`) filling the
  detail column.
- Empty-conversation state: centered "No messages yet — say hello"
  (Subheadline, `text.secondary`), composer still active.
- Composer bar: pinned to bottom, `bg.surface2`, 1px top hairline, 12pt
  internal padding: text field (`bg.surface`, radius 10pt, flex width) +
  8pt gap + circular send button (44×44pt, `accent` fill, arrow-up
  glyph, disabled/35%-opacity when input empty, and reachable via
  Return — Shift+Return inserts a newline instead of sending). The text
  field starts at the same height as the send button (28pt inner text
  area, ~44pt bar height including padding) rather than a taller fixed
  minimum, and grows upward only as far as needed for the current text,
  capped at 120pt, with no visible scroll indicator inside it even once
  it's scrolling internally past the cap.

### B2 — Message bubble (component)
- Radius 14pt with one corner reduced to 4pt on the "tail" side (bottom-
  trailing for sent, bottom-leading for received) — the one asymmetric
  detail that reads as direction at a glance, on top of alignment/color.
- Sent: right-aligned, `accent` fill, `on-accent` text.
- Received: left-aligned, `bg.surface` fill, 1px `border.hairline`,
  `text.primary` text.
- Padding 12pt horizontal, 8pt vertical. Body text style, `text-wrap:
  pretty` equivalent (native line-break strategy, avoid orphans).
- Timestamp: Caption, 60% opacity of the bubble's text color, right-
  aligned below the bubble (own line, 2pt gap) rather than inside it —
  keeps copy-to-clipboard scoped to the message text only.
- Copy affordance: long-press (iOS) / right-click or hover-reveal
  (macOS) → "Copy" in a context menu, not a permanent visible icon per
  bubble (keeps the transcript visually quiet).

### B3 — Add contact (sheet/dialog)
- Title "Add contact" (Title). Fields, 16pt stack gap:
  1. "Contact code" — multi-line-capable text field (codes are long),
     `bg.surface2`, radius 10pt, min height 80pt, Mono text, paste-
     friendly (auto-trim whitespace on paste).
  2. "Name (optional)" — single-line field, 44pt height.
- Inline error under field 1 on invalid/malformed code (`danger`
  Caption): "That doesn't look like a valid contact code."
- Actions: trailing-aligned pair, "Cancel" (plain) + "Add" (accent,
  filled, disabled until field 1 has content). Sheet width 440pt on
  macOS — a **fixed** width, not a min/ideal range, since letting
  SwiftUI negotiate the width caused a visible pop/resize right after
  the sheet finished presenting. On iOS/iPadOS the sheet uses a single
  `.large` detent (not `[.medium, .large]`) so it always opens fully
  expanded instead of landing at the half-height detent first. This
  fixed-width/single-detent treatment applies to all of B3–B6.
- The "Contact code" placeholder text ("Paste a contact code…") does not
  inherit the sheet's presentation-spring animation, so it doesn't
  visibly lag behind the field during the opening transition.

### B4 — Edit contact (sheet/dialog)
- Title "Edit contact". 64×64pt avatar centered at top (16pt below
  title), same deterministic-hue placeholder as list rows.
- "Name" field, single-line, 44pt, autofocused.
- Read-only address block directly below: `bg.surface2`, radius 10pt,
  12pt padding, Mono/Caption text at `text.primary` (full contrast, not
  disabled-styled), truncated-middle format, with a small "copy" icon
  trailing inside the block.
- Actions: "Cancel" / "Save" (accent), same layout as B3. Sheet width
  360pt.

### B5 — My contact code (sheet/dialog)
- Title "My code". Explanatory line under title (Subheadline,
  `text.secondary`): "Share this so others can message you."
- Code block: same treatment as A2's phrase card — `bg.surface2`, radius
  20pt, 20pt padding, Mono/Body `text.primary`, full contrast, word-
  wrapped (codes are one long string; break at fixed character intervals
  e.g. every 4 chars with a hairline-thin visual grouping, not raw
  unbroken wrap).
- "Copy to clipboard" — full-width button directly under the block,
  bordered/secondary style (not `accent`-filled — this is a supporting
  action, not the sheet's primary CTA in the A1-primary sense), 44pt
  height, radius 10pt. On tap: label swaps to "Copied" + checkmark for
  1.5s, then reverts.
- Divider (hairline, 16pt vertical margin), then "Your address" label
  (Caption, `text.secondary`) + the raw address in Mono/Caption,
  `text.primary`, its own smaller copy icon.
- "Close" as the sheet's dismiss (nav-bar Done on iOS, standard sheet
  close on macOS) rather than a second full-width button.

### B6 — Network settings (sheet/dialog)
- Title "Network settings". Grouped-list style (like iOS Settings):
  section 1 "Gateways" — "Bootstrap gateways" multi-line field
  (comma-separated, Mono, min height 80pt) with Caption helper text below
  ("Comma-separated host:port list. Leave blank to use the built-in
  defaults.").
- Section 2 "Certificate" — platform-specific, since only macOS has a
  meaningful notion of a typed filesystem path: on macOS, a "CA
  certificate path" single-line field + a trailing "Choose file…" button
  (native file picker), either one can set the value. On iOS/iPadOS
  there's no editable path field at all — just the picked file's name
  (or "No file selected") as plain text next to the "Choose file…"
  button, since the file picker is the only way a path can ever get set
  there.
- Section 3, visually separated (extra 24pt gap + its own subtle
  `warning`-tinted background strip, radius 10pt, 12pt padding) —
  "Skip TLS verification" toggle with Caption warning copy directly
  under it: "Only for local/dev networks. Leave off otherwise." This
  section must look distinct from sections 1–2, not just another row.
- Actions: "Cancel" / "Save". On save, a non-blocking banner/alert:
  "Restart OpenChat to apply network changes" (`warning` tinted, dismiss
  or auto-hide after ~4s) — not a silent save.

### B7 — Log out (confirmation)
- System-native confirmation pattern: macOS `NSAlert`-style / iOS
  `.confirmationDialog` or `.alert`, not a custom full sheet — this
  should feel like the platform's own "are you sure" idiom.
- Title: "Log out of OpenChat?" Body: plain-language statement of what's
  destroyed ("This deletes your wallet and message history from this
  device. Your recovery phrase is the only way back in — make sure
  you've saved it.") — `text.primary`, no separate warning-icon block
  needed here (the platform alert chrome already signals severity).
- Actions: "Cancel" (default/focused) + "Log Out" styled as the
  destructive action (`danger` text/fill per platform convention, e.g.
  iOS `role: .destructive`).

---

## 4. Iconography

Apple: SF Symbols. Suggested symbol per action (final pick left to
implementation, must stay consistent once chosen):

| Action | SF Symbol |
|---|---|
| Add contact | `person.badge.plus` |
| My code | `qrcode` or `person.crop.circle.badge.checkmark` |
| Settings | `gearshape` |
| Log out | `rectangle.portrait.and.arrow.right` |
| Send | `arrow.up.circle.fill` |
| Edit contact | `pencil` |
| Copy | `doc.on.doc` |

Windows: Fluent System Icons (equivalent glyphs). Linux: platform icon
theme (Adwaita/Breeze) equivalents. Android: Material Symbols
equivalents. Same seven actions, no new ones.

---

## 5. States checklist (per interactive surface)

Every field/button/list in §3 needs, at minimum: **default, focused,
disabled, error** (fields) and **default, pressed, disabled** (buttons).
Loading states are narrow and explicit by design (per functional spec,
no blocking spinners): the only ongoing-async indicator is B1's status
strip ("connecting…" / "syncing…"); everything else either completes
fast enough not to need one or shows its result/error immediately
(add contact, import phrase, unlock).

---

## 6. Open questions for implementation

- A4/A5 PIN: numeric-only vs. alphanumeric — affects keyboard type and
  whether "4+ characters" means digits or any characters.
- B1 "most-recently-active ordering": not implemented server-side yet;
  native build should default to insertion order until that lands, per
  functional spec.
- macOS system-accent-color mode (using the user's own macOS accent
  instead of the fixed indigo) is a nice-to-have, not required for v1.
