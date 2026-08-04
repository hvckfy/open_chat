//
//  appApp.swift
//  app
//
//  Created by Руслан on 04.08.2026.
//

import SwiftUI
import OpenChatKit

// Single unified (iOS/iPadOS/macOS) app target — see
// ../../README.md "Setting up the Xcode project". All real UI lives in
// OpenChatKit's RootView; this file only adds macOS's window geometry
// (design-spec.md §2.1: min 880×560, default 1120×720) and drops the
// default "New Window" menu item, since OpenChat is a single-window app.
@main
struct appApp: App {
    var body: some Scene {
        WindowGroup {
            RootView()
                #if os(macOS)
                .frame(minWidth: 880, minHeight: 560)
                .frame(idealWidth: 1120, idealHeight: 720)
                #endif
        }
        #if os(macOS)
        .windowResizability(.contentSize)
        .commands {
            CommandGroup(replacing: .newItem) {}
        }
        #endif
    }
}
