import SwiftUI

/// Design tokens from `docs/design-spec.md` §1 — colors (precomputed
/// from the spec's OKLCH values into sRGB, see Theme+Color.swift),
/// typography (SwiftUI text-style mapping from §1.2), and spacing/
/// geometry (the 8pt grid from §1.3). Every screen/component in this
/// package reads sizes and colors from here, not literals, so retuning
/// the palette or grid later touches one file.
enum Theme {

    // MARK: - Color (§1.1)

    enum Color {
        static let bgCanvas = SwiftUI.Color(
            light: (250 / 255, 248 / 255, 245 / 255, 1),
            dark: (15 / 255, 13 / 255, 10 / 255, 1))

        static let bgSurface = SwiftUI.Color(
            light: (255 / 255, 255 / 255, 253 / 255, 1),
            dark: (26 / 255, 24 / 255, 21 / 255, 1))

        static let bgSurface2 = SwiftUI.Color(
            light: (242 / 255, 240 / 255, 236 / 255, 1),
            dark: (38 / 255, 36 / 255, 32 / 255, 1))

        static let bgOverlay = SwiftUI.Color(
            light: (0, 0, 0, 0.32),
            dark: (0, 0, 0, 0.55))

        static let borderHairline = SwiftUI.Color(
            light: (217 / 255, 215 / 255, 211 / 255, 1),
            dark: (53 / 255, 50 / 255, 46 / 255, 1))

        static let textPrimary = SwiftUI.Color(
            light: (28 / 255, 26 / 255, 23 / 255, 1),
            dark: (241 / 255, 238 / 255, 234 / 255, 1))

        static let textSecondary = SwiftUI.Color(
            light: (95 / 255, 93 / 255, 90 / 255, 1),
            dark: (154 / 255, 152 / 255, 148 / 255, 1))

        static let textTertiary = SwiftUI.Color(
            light: (136 / 255, 134 / 255, 130 / 255, 1),
            dark: (112 / 255, 110 / 255, 107 / 255, 1))

        static let accent = SwiftUI.Color(
            light: (70 / 255, 108 / 255, 200 / 255, 1),
            dark: (118 / 255, 161 / 255, 255 / 255, 1))

        static let accentPressed = SwiftUI.Color(
            light: (48 / 255, 84 / 255, 174 / 255, 1),
            dark: (95 / 255, 135 / 255, 231 / 255, 1))

        static let onAccent = SwiftUI.Color(
            light: (254 / 255, 251 / 255, 248 / 255, 1),
            dark: (6 / 255, 9 / 255, 17 / 255, 1))

        static let danger = SwiftUI.Color(
            light: (193 / 255, 60 / 255, 59 / 255, 1),
            dark: (242 / 255, 113 / 255, 106 / 255, 1))

        static let success = SwiftUI.Color(
            light: (52 / 255, 143 / 255, 79 / 255, 1),
            dark: (92 / 255, 181 / 255, 114 / 255, 1))

        static let warning = SwiftUI.Color(
            light: (220 / 255, 163 / 255, 49 / 255, 1),
            dark: (227 / 255, 173 / 255, 75 / 255, 1))
    }

    // MARK: - Typography (§1.2)

    enum Font {
        /// A1 wordmark, A2 screen title.
        static let display = SwiftUI.Font.largeTitle.weight(.bold)
        /// Screen/dialog titles (A3–A5, B3–B7).
        static let title = SwiftUI.Font.title2.weight(.semibold)
        /// Contact name in list row, message-sender emphasis.
        static let headline = SwiftUI.Font.headline.weight(.semibold)
        /// Message text, input fields, body copy.
        static let body = SwiftUI.Font.body
        /// Secondary row text, dialog explanatory copy.
        static let subheadline = SwiftUI.Font.subheadline
        /// Timestamps, status indicator, field hints.
        static let caption = SwiftUI.Font.caption
        /// Recovery phrase, contact code, addresses, hex keys.
        static let mono = SwiftUI.Font.body.monospaced()
        static let monoCaption = SwiftUI.Font.caption.monospaced()
    }

    // MARK: - Spacing & geometry (§1.3)

    enum Spacing {
        /// Screen/dialog outer margin — 24pt macOS/iPad.
        static let screenMarginRegular: CGFloat = 24
        /// Screen/dialog outer margin — 20pt iPhone.
        static let screenMarginCompact: CGFloat = 20
        /// Stack spacing between grouped elements.
        static let group: CGFloat = 16
        /// Spacing within a tight group (label + field).
        static let tight: CGFloat = 8
    }

    enum Size {
        /// Standard control height (buttons, list rows, text fields) —
        /// also the minimum hit target on every platform.
        static let control: CGFloat = 44
        /// A1/A2/A4/A5's emphasized primary action button.
        static let primaryButton: CGFloat = 50
        static let avatarRow: CGFloat = 40
        static let avatarDialog: CGFloat = 64
        static let pinKeyDiameterCompact: CGFloat = 60
        static let pinKeyDiameterRegular: CGFloat = 52
        static let sidebarIdeal: CGFloat = 280
        static let sidebarMin: CGFloat = 220
        static let sidebarMax: CGFloat = 360
        static let windowMinWidth: CGFloat = 880
        static let windowMinHeight: CGFloat = 560
        static let windowDefaultWidth: CGFloat = 1120
        static let windowDefaultHeight: CGFloat = 720
    }

    enum Radius {
        /// Buttons, text fields, small cards.
        static let small: CGFloat = 10
        /// Message bubbles, sheet/dialog panels.
        static let medium: CGFloat = 14
        /// Large emphasis cards (A2/B5's phrase/code card).
        static let large: CGFloat = 20
        /// The "tail" corner of a message bubble (§B2).
        static let bubbleTail: CGFloat = 4
    }
}
