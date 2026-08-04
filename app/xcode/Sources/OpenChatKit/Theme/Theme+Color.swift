import SwiftUI

#if canImport(UIKit)
import UIKit
#elseif canImport(AppKit)
import AppKit
#endif

/// Cross-platform "dynamic color" helper: builds a `Color` that swaps
/// between a light and a dark sRGB value at *render* time (following the
/// system/window appearance), the same way an Xcode asset-catalog color
/// set with light/dark variants would — without needing an asset
/// catalog, since this package has no Xcode project of its own to hold
/// one. Values are precomputed sRGB (see the comment on `Theme.Color`)
/// rather than computed from OKLCH at runtime: SwiftUI has no built-in
/// OKLCH initializer, and converting once, offline, against the exact
/// numbers in design-spec.md §1.1 is both simpler and avoids shipping a
/// color-space conversion just for this.
extension Color {
    init(light: (r: Double, g: Double, b: Double, a: Double), dark: (r: Double, g: Double, b: Double, a: Double)) {
        #if canImport(UIKit)
        self.init(UIColor { trait in
            trait.userInterfaceStyle == .dark
                ? UIColor(red: dark.r, green: dark.g, blue: dark.b, alpha: dark.a)
                : UIColor(red: light.r, green: light.g, blue: light.b, alpha: light.a)
        })
        #elseif canImport(AppKit)
        self.init(NSColor(name: nil, dynamicProvider: { appearance in
            let isDark = appearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
            let v = isDark ? dark : light
            return NSColor(red: v.r, green: v.g, blue: v.b, alpha: v.a)
        }))
        #else
        self.init(red: light.r, green: light.g, blue: light.b, opacity: light.a)
        #endif
    }
}
