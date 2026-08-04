module openchat/fyne

go 1.22

require (
	fyne.io/fyne/v2 v2.6.0
	golang.org/x/crypto v0.33.0
	openchat v0.0.0-00010101000000-000000000000
)

// openchat is the backend/core module at ../golang (validator/relay node,
// pkg/client, pkg/crypto — everything this GUI reuses unmodified). It's
// not published anywhere, so this replace directive is load-bearing, not
// optional: without it, `openchat v0.0.0-...` above can't resolve.
replace openchat => ../golang

// Run `go mod tidy` here once (needs network access to the Go module
// proxy) to populate go.sum with Fyne's full transitive dependency tree
// — that tree is exactly what moving this module out of app/golang was
// for: it no longer touches the backend's go.sum/Docker build context at
// all.
