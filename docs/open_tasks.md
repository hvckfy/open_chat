# Open tasks — next stage (client-only)

None of the tasks below require changes to the protocol, consensus, or
transaction format — only client-side logic (`app/golang/mobile`,
`pkg/client`, `app/xcode`). That's deliberate: this network has no
pruning (every validator keeps every transaction in BadgerDB forever),
so any change to the wire format is a change forever, for everyone. Both
tasks below are shaped specifically to avoid needing one.

## 1. Own-profile avatar (local only, never touches the server)

Right now there's no concept of an "avatar" anywhere — `ContactAvatar.swift`
draws a colored circle with an initial letter (the color is a hash of the
address); there are no pictures at all. That should stay true at the
protocol level: an avatar is purely local and should never cross the
network (unlike attachments in task 3, this isn't worth spending chain
bytes on).

- New local store (same pattern as `ChatStore.swift` — a file in
  Application Support, not Keychain; a picture isn't a secret): a file
  path or `Data` for the user's own avatar.
- UI entry point: `Dialogs/MyCodeSheet.swift` (B5, "My code") is the
  natural place — add a tap on the avatar circle there that opens a
  photo picker (`PhotosPicker` on iOS, `NSOpenPanel`/`.fileImporter` on
  macOS — `NetworkSettingsSheet.swift` already has the same `#if
  os(macOS)` split for the CA certificate picker; that pattern can be
  copied).
- `Components/ContactAvatar.swift` — add an optional image parameter so
  it draws the picture instead of the initial when one is set.

## 2. Contact avatars (same mechanism, local only)

A natural extension of task 1, reusing the same picker:

- `Models/Contact.swift`: add `var avatarFileName: String?`.
- `Dialogs/EditContactSheet.swift` (and possibly `AddContactSheet.swift`):
  add the same picker.
- `ContactAvatar.swift` draws the contact's picture when one is set.

Important: there is no network round-trip to fetch a peer's avatar, and
there shouldn't be one — only whatever the user has locally assigned to
that contact themselves. That's a deliberate protocol boundary, not a
missing feature.

## 3. Attachments (a file is just bytes inside the already-encrypted message)

Exactly what was proposed: `Client.SendMessage` in
`pkg/client/client.go:116` already takes `plaintext []byte`, not a
string — the transport/encryption layer already supports an attachment
with zero changes there. All that changes is what goes into those bytes,
and how it's parsed back out on the other end.

Payload format (JSON, encoded into the same `[]byte` that used to carry
plain text):
```json
{
  "message": "message text",
  "attach": "<base64 file bytes>",
  "filename": "photo.jpg",
  "mime": "image/jpeg"
}
```

Where exactly to change things:

- **Sending** — `mobile/session.go`: next to the existing `SendText`
  (around line 122, a one-line `[]byte(text)` wrapper), add
  `SendAttachment(toAddress, toX25519Hex, text string, data []byte,
  filename, mime string) (string, error)` that marshals the JSON above
  and calls the same `s.client.SendMessage(...)`. A new method, zero
  changes in `pkg/client` or the node.
- **Receiving — two spots**, both of which currently treat decrypted
  bytes unconditionally as plain text:
  - `mobile/history.go:75` (`Text: string(ev.Plaintext)` inside
    `FetchHistory`);
  - `mobile/session.go:159` (`Listen`, where `m.Plaintext` goes straight
    into `MessageListener.OnMessage`).

  In both, try `json.Unmarshal` into the payload shape above first; if
  it parses and matches that shape, surface `message`/`attach`
  separately; otherwise fall back to today's behavior (plain text).
  That keeps backward compatibility: older messages, and messages from
  a client that hasn't shipped this feature yet, keep displaying
  exactly as before. Best to factor the parsing into one shared helper
  rather than duplicating it in both places.
- **Swift side**: `Models/StoredMessage.swift` — add
  `attachmentBase64: String?` / `attachmentFileName: String?` /
  `attachmentMime: String?`; `Core/LiveCore.swift` — decode the new
  fields from Go; `Components/MessageBubble.swift` — show an image
  preview (or a plain filename chip for non-image data);
  `Main/ConversationView.swift` — an "attach" button next to the send
  field (`PhotosPicker`/`.fileImporter`).

## 4. Client-side attachment size cap

An important caveat on task 3: since an attachment rides inside the same
ciphertext as a normal message, and this network never prunes anything,
one accidentally-sent 20MB PNG stays permanently in every validator's
database forever. Needs a client-side limit before sending: downscale +
compress photos to some reasonable max (a few hundred KB), and an
explicit warning/rejection in the UI if the file is still too big after
compression. A separate task from 3 on purpose, so it doesn't block that
work if the size limit needs tuning later.

## 5. Long-press on a message: copy / save attachment

`Components/MessageBubble.swift` — a context menu (`.contextMenu`):
"Copy text" for the text, "Save to Files/Photos" for the attachment
(once task 3 exists). Pure UI, doesn't touch the Go core at all.

## 6. Local search across chat history

The full history already lives entirely in `ChatStore.swift` locally
(a JSON file, `messagesByAddress`) — nothing currently lets you filter
it. Add a search field in `Main/ChatListView.swift` that filters by
contact name/address and/or message text across all conversations.

## 7. Export chat history

The app currently has no backup story at all — `ChatStore`'s own comment
says outright "a single JSON file... a killed app never loses a
message," but that's the only safety net if a phone is lost or replaced.
Add an "Export chat history" action (alongside Log out/Network
settings) that shares the `chats.v1.json` file (or a copy of it) via the
system share sheet.

## 8. Pin / mute a conversation in the list

Purely local metadata on the contact (`Contact.swift`: `isPinned: Bool`,
`isMuted: Bool`), with sort/filter logic in `Main/ChatListView.swift`.
Needs nothing new on the Go side or in the protocol.
