// Reference source for the iOS/iPadOS App target — NOT part of the
// OpenChatKit Swift package; see the macOS counterpart's header comment
// for why, and ../../README.md "Setting up the Xcode project" for where
// this actually goes.
//
// No window-geometry setup needed here (unlike macOS) — safe-area and
// size-class handling for iPhone vs. iPad both live inside
// OpenChatKit's MainSplitView (design-spec.md §2.2).

import SwiftUI
import OpenChatKit

@main
struct OpenChatIOSApp: App {
    var body: some Scene {
        WindowGroup {
            RootView()
        }
    }
}
