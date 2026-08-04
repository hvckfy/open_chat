import Foundation
import os

/// Shared `os.Logger` instances, one per area, so output can be filtered
/// by category in Console.app (Console → search "network.openchat.app",
/// or `log stream --predicate 'subsystem == "network.openchat.app"'` in
/// Terminal — works for both the Simulator and a connected device).
///
/// Added specifically to diagnose "chat updates aren't happening"
/// reports: every point where a message enters or leaves `ChatStore`,
/// and every state change in the connect/history-sync/live-listen
/// pipeline that could explain why one hasn't shown up, gets a line
/// here — connect/sync/listen starting and finishing, every message
/// stored (or a history one skipped as a dupe), every status change.
///
/// Deliberately never logs message *text* — only metadata (address,
/// tx hash, byte counts, counts). These are end-to-end encrypted
/// private messages; the whole point of that is defeated if the
/// plaintext ends up sitting in the system log instead.
enum Log {
    static let session = Logger(subsystem: "network.openchat.app", category: "session")
    static let chatStore = Logger(subsystem: "network.openchat.app", category: "chatstore")
}
