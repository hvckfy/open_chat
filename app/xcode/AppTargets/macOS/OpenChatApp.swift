// Reference source for the macOS App target — NOT part of the
// OpenChatKit Swift package (SwiftPM can only build one product, a
// library; `@main` app entry points live in the actual Xcode project's
// targets). Copy this file's content into the macOS target Xcode
// creates for you (see ../../README.md "Setting up the Xcode project",
// step 4) rather than trying to build this file directly.
//
// Everything behind the window itself — onboarding, the chat UI, all
// state — lives in OpenChatKit's RootView; this file only sets the
// window geometry design-spec.md §2.1 asks for (min 880×560, default
// 1120×720) and removes the default "New Window" menu item (single-
// window app).

import SwiftUI
import OpenChatKit

@main
struct OpenChatMacApp: App {
    var body: some Scene {
        WindowGroup {
            RootView()
                .frame(minWidth: 880, minHeight: 560)
                .frame(idealWidth: 1120, idealHeight: 720)
        }
        .windowResizability(.contentSize)
        .commands {
            CommandGroup(replacing: .newItem) {}
        }
    }
}
