package app

import "openchat/pkg/client"

// Deprecated: this logic moved to pkg/client (app/golang/pkg/client/
// contactcode.go) so the CLI, this Fyne app, and the mobile bridge
// (app/golang/mobile) all share one implementation instead of each having
// their own copy. These are kept as thin re-exports — rather than removed
// with every call site updated to say client.MyContactCode /
// client.ParseContactCode directly — only because this sandbox's
// filesystem mount won't let this file be deleted outright. New code
// should call the pkg/client versions directly instead of importing these.

// ParsedContact is what ParseContactCode extracts from a pasted code.
//
// Deprecated: use client.ParsedContact.
type ParsedContact = client.ParsedContact

// MyContactCode returns a single shareable string encoding both keys a
// contact needs to message this wallet.
//
// Deprecated: use client.MyContactCode.
func MyContactCode(w *client.Wallet) string { return client.MyContactCode(w) }

// ParseContactCode reverses MyContactCode.
//
// Deprecated: use client.ParseContactCode.
func ParseContactCode(code string) (*ParsedContact, error) { return client.ParseContactCode(code) }
