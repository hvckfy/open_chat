import Foundation

/// Selects which `OpenChatCore` implementation the rest of the app uses.
/// Defaults to the real Go core when the generated framework is linked
/// (see LiveCore.swift), and falls back to MockCore otherwise — so
/// SwiftUI previews, unit tests, and this package's own `swift build`
/// (which never links an Xcode-only binary framework) all just work
/// without conditional code anywhere else.
enum OpenChatCoreProvider {
    #if canImport(OpenChatMobile)
    static let current: OpenChatCore.Type = LiveCore.self
    #else
    static let current: OpenChatCore.Type = MockCore.self
    #endif
}
