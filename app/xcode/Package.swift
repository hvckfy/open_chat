// swift-tools-version: 5.9
import PackageDescription

// OpenChatKit is a Swift package, not the app itself: it holds every
// piece of UI and view-model code shared between the macOS and iOS app
// targets (design-spec.md explicitly shares one token/component set
// across platforms, just wired into different navigation shells). The
// actual App targets — with their Info.plist, app icons, signing,
// entitlements, and the generated OpenChatMobile.xcframework — live in
// an Xcode project that adds this package as a local dependency; see
// ../README.md "Setting up the Xcode project" for the exact steps, since
// that project itself isn't something this sandbox can generate (it
// needs Xcode's own project format and can't be safely hand-written).
let package = Package(
    name: "OpenChatKit",
    platforms: [
        .iOS(.v17),
        .macOS(.v14),
    ],
    products: [
        .library(name: "OpenChatKit", targets: ["OpenChatKit"]),
    ],
    targets: [
        .target(
            name: "OpenChatKit",
            path: "Sources/OpenChatKit"
        ),
        // No .testTarget here: one was declared pointing at
        // Tests/OpenChatKitTests, but that directory was never created
        // (nothing under it in git history either) — an invalid custom
        // `path` on a target is a hard manifest error, so it broke
        // `swift build`/`swift test` and Xcode's resolution of this
        // package entirely, not just "no tests ran". Re-add a real
        // .testTarget (with an actual Tests/OpenChatKitTests directory
        // containing at least one test file) when there's something to
        // put in it.
    ]
)
