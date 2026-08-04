import SwiftUI

/// design-code.md's "Global structure" main-app state, wired to the
/// platform shells design-spec.md §2 calls for: `NavigationSplitView`
/// on macOS/iPad, `NavigationStack` on iPhone. Both push the same
/// ChatListView/ConversationView pair (§B1/§B2) — only the container
/// differs. Owns presentation of the B3–B7 dialogs, since every one of
/// them is reached from this screen's toolbar/rows.
// Internal, not public: AppModel/SessionModel are themselves internal
// (this package's only public entry point is RootView), so this can't be
// public anyway — a public initializer can't take internal-type
// parameters. Only ever constructed from RootView.swift, in-module.
struct MainSplitView: View {
    @ObservedObject var appModel: AppModel
    @ObservedObject var session: SessionModel

    @State private var selection: Contact?
    @State private var activeSheet: MainSheet?
    @State private var showLogoutConfirm = false

    // Only iPad still gets a collapsible sidebar (system-managed, via
    // this binding + its own default toggle). macOS doesn't use this at
    // all anymore — see splitLayout's comment for why.
    #if os(iOS)
    @State private var columnVisibility: NavigationSplitViewVisibility = .all
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    #endif

    init(appModel: AppModel, session: SessionModel) {
        self.appModel = appModel
        self.session = session
    }

    var body: some View {
        #if os(macOS)
        splitLayout
        #else
        // iPhone: compact width → NavigationStack. iPad: regular width →
        // NavigationSplitView, same structure as macOS (design-spec.md §2.2).
        if horizontalSizeClass == .compact {
            stackLayout
        } else {
            splitLayout
        }
        #endif
    }

    // MARK: - macOS / iPad: NavigationSplitView (design-spec.md §2.1/§2.2 iPad)

    @ViewBuilder
    private var sidebarContent: some View {
        ChatListView(session: session, chatStore: session.chatStore, selection: $selection, onAddContact: { activeSheet = .addContact }, onEditContact: { activeSheet = .editContact($0) })
    }

    @ViewBuilder
    private var detailContent: some View {
        if let selection {
            ConversationView(session: session, chatStore: session.chatStore, contact: selection, onEdit: { activeSheet = .editContact(selection) })
        } else {
            NoSelectionView()
        }
    }

    private var splitLayout: some View {
        #if os(macOS)
        // Plain `HSplitView`, not `NavigationSplitView`, on macOS.
        //
        // `NavigationSplitView` always tries to inject its own default
        // sidebar-toggle toolbar button, and every way of dealing with
        // that made things worse, not better, across three separate
        // attempts:
        //   1. A custom "open-sided chevron" meant to replace the system
        //      glyph via `.toolbar(removing: .sidebarToggle)` + a
        //      matching `ToolbarItem` — the system glyph kept showing up
        //      duplicated alongside the custom one no matter where
        //      exactly the two modifiers were attached.
        //   2. The plain system toggle left alone (no custom button,
        //      `columnVisibility` bound but otherwise untouched) — the
        //      sidebar then rendered as a separate floating/overlaid
        //      panel (visible desktop wallpaper in the gap around it)
        //      instead of a normal in-window collapse.
        //   3. No `columnVisibility` binding at all, plus
        //      `.toolbar(removing: .sidebarToggle)` with no replacement
        //      button — the system button *still* showed up regardless.
        // `.toolbar(removing: .sidebarToggle)` has proven completely
        // unreliable here, so this sidesteps the whole mechanism: an
        // `HSplitView` has no "sidebar" concept and no associated
        // default toolbar item, so there's nothing left for the system
        // to inject in the first place. Still two resizable columns with
        // a draggable divider (`HSplitView`'s native behavior, driven by
        // each child's own `.frame(minWidth:idealWidth:maxWidth:)`) and
        // the same window-unified `.toolbar` buttons — just without
        // `NavigationSplitView`'s toolbar-injection baggage.
        HSplitView {
            sidebarContent
                .frame(minWidth: Theme.Size.sidebarMin, idealWidth: Theme.Size.sidebarIdeal, maxWidth: Theme.Size.sidebarMax)
                .frame(maxHeight: .infinity)
            detailContent
                .frame(minWidth: 480, maxWidth: .infinity, maxHeight: .infinity)
        }
        .toolbar { toolbarContent }
        .sheet(item: $activeSheet) { sheet(for: $0) }
        .logoutConfirmation(isPresented: $showLogoutConfirm, onConfirm: appModel.logout)
        #else
        NavigationSplitView(columnVisibility: $columnVisibility) {
            sidebarContent
                .navigationSplitViewColumnWidth(min: Theme.Size.sidebarMin, ideal: Theme.Size.sidebarIdeal, max: Theme.Size.sidebarMax)
        } detail: {
            detailContent
        }
        .toolbar { toolbarContent }
        .sheet(item: $activeSheet) { sheet(for: $0) }
        .logoutConfirmation(isPresented: $showLogoutConfirm, onConfirm: appModel.logout)
        #endif
    }

    // MARK: - iPhone: NavigationStack (design-spec.md §2.2)

    private var stackLayout: some View {
        NavigationStack {
            ChatListView(session: session, chatStore: session.chatStore, selection: $selection, asNavigationLinks: true, onAddContact: { activeSheet = .addContact }, onEditContact: { activeSheet = .editContact($0) })
                .navigationTitle("Chats")
                #if os(iOS)
                // Fixed/inline nav bar instead of the default large-title,
                // which collapses as the list scrolls — that collapse
                // animation was what made the "Chats" title and the
                // status strip directly under it look like they were
                // drifting apart / scrolling independently. iOS/iPadOS-only
                // API — stackLayout is only ever *used* on those platforms
                // (see body's #if os(macOS) above), but Swift still type-
                // checks this property's body unconditionally on every
                // platform, so the modifier itself still needs its own
                // #if os(iOS) guard or a macOS build fails to compile.
                .navigationBarTitleDisplayMode(.inline)
                #endif
                .toolbar { toolbarContent }
                .navigationDestination(for: Contact.self) { contact in
                    ConversationView(session: session, chatStore: session.chatStore, contact: contact, onEdit: { activeSheet = .editContact(contact) })
                }
        }
        .sheet(item: $activeSheet) { sheet(for: $0) }
        .logoutConfirmation(isPresented: $showLogoutConfirm, onConfirm: appModel.logout)
    }

    // MARK: - Toolbar (design-spec.md §2.1/§2.2: Add contact, My code, Settings; Log out in overflow)

    @ToolbarContentBuilder
    private var toolbarContent: some ToolbarContent {
        #if os(iOS)
        ToolbarItem(placement: .topBarTrailing) {
            Button { activeSheet = .addContact } label: { Image(systemName: "person.badge.plus") }
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button { activeSheet = .myCode } label: { Image(systemName: "qrcode") }
        }
        ToolbarItem(placement: .topBarTrailing) {
            Menu {
                Button { activeSheet = .settings } label: { Label("Network settings", systemImage: "gearshape") }
                Button(role: .destructive) { showLogoutConfirm = true } label: { Label("Log out", systemImage: "rectangle.portrait.and.arrow.right") }
            } label: {
                Image(systemName: "ellipsis.circle")
            }
        }
        #else
        ToolbarItem { Button { activeSheet = .addContact } label: { Image(systemName: "person.badge.plus") } }
        ToolbarItem { Button { activeSheet = .myCode } label: { Image(systemName: "qrcode") } }
        ToolbarItem { Button { activeSheet = .settings } label: { Image(systemName: "gearshape") } }
        ToolbarItem {
            Menu {
                Button(role: .destructive) { showLogoutConfirm = true } label: { Label("Log out", systemImage: "rectangle.portrait.and.arrow.right") }
            } label: {
                Image(systemName: "ellipsis.circle")
            }
        }
        #endif
    }

    // MARK: - B3–B6 sheets

    @ViewBuilder
    private func sheet(for item: MainSheet) -> some View {
        switch item {
        case .addContact:
            AddContactSheet(core: appModel.core, chatStore: session.chatStore, onAdded: { selection = $0 })
        case .editContact(let contact):
            EditContactSheet(contact: contact, chatStore: session.chatStore)
        case .myCode:
            MyCodeSheet(contactCode: session.myContactCode, address: session.myAddress)
        case .settings:
            NetworkSettingsSheet(initial: appModel.networkSettings, onSave: appModel.saveNetworkSettings)
        }
    }
}

private enum MainSheet: Identifiable, Equatable {
    case addContact
    case editContact(Contact)
    case myCode
    case settings

    var id: String {
        switch self {
        case .addContact: return "addContact"
        case .editContact(let c): return "editContact:\(c.address)"
        case .myCode: return "myCode"
        case .settings: return "settings"
        }
    }
}
