package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"openchat/pkg/client"
)

// Network settings are a handful of Fyne Preferences entries — simpler
// than a JSON file, and they're the idiomatic cross-platform (desktop +
// mobile) key/value store Fyne already provides.
const (
	prefBootstrap = "network.bootstrap"
	prefCAPath    = "network.ca_path"
	prefInsecure  = "network.insecure_skip_verify"
)

type networkSettings struct {
	Bootstrap []string
	CAPath    string
	Insecure  bool
}

func loadNetworkSettings(a fyne.App) networkSettings {
	p := a.Preferences()
	csv := p.StringWithFallback(prefBootstrap, "")
	var bootstrap []string
	if csv != "" {
		for _, s := range strings.Split(csv, ",") {
			if s = strings.TrimSpace(s); s != "" {
				bootstrap = append(bootstrap, s)
			}
		}
	}
	if len(bootstrap) == 0 {
		bootstrap = client.DefaultBootstrapGateways
	}
	return networkSettings{
		Bootstrap: bootstrap,
		CAPath:    p.String(prefCAPath),
		Insecure:  p.Bool(prefInsecure),
	}
}

func saveNetworkSettings(a fyne.App, s networkSettings) {
	p := a.Preferences()
	p.SetString(prefBootstrap, strings.Join(s.Bootstrap, ","))
	p.SetString(prefCAPath, s.CAPath)
	p.SetBool(prefInsecure, s.Insecure)
}

// showNetworkSettingsDialog lets the user point the app at their own
// OpenChat network (e.g. "localhost:9091,localhost:9092" against the
// docker-compose demo, with a CA cert path, or "insecure" while testing
// with self-signed certs).
func showNetworkSettingsDialog(win fyne.Window, onSaved func(networkSettings)) {
	cur := loadNetworkSettings(fyne.CurrentApp())

	bootstrapEntry := widget.NewEntry()
	bootstrapEntry.SetPlaceHolder("gateway1.example.com:9090,gateway2.example.com:9090")
	bootstrapEntry.SetText(strings.Join(cur.Bootstrap, ","))

	caEntry := widget.NewEntry()
	caEntry.SetPlaceHolder("(optional) path to network CA cert .pem")
	caEntry.SetText(cur.CAPath)

	insecureCheck := widget.NewCheck("Skip TLS verification (only for local/dev self-signed certs)", nil)
	insecureCheck.SetChecked(cur.Insecure)

	form := widget.NewForm(
		widget.NewFormItem("Bootstrap gateways", bootstrapEntry),
		widget.NewFormItem("CA certificate", caEntry),
		widget.NewFormItem("", insecureCheck),
	)

	d := dialog.NewCustomConfirm("Network settings", "Save", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		var list []string
		for _, s := range strings.Split(bootstrapEntry.Text, ",") {
			if s = strings.TrimSpace(s); s != "" {
				list = append(list, s)
			}
		}
		next := networkSettings{Bootstrap: list, CAPath: strings.TrimSpace(caEntry.Text), Insecure: insecureCheck.Checked}
		saveNetworkSettings(fyne.CurrentApp(), next)
		if onSaved != nil {
			onSaved(next)
		}
	}, win)
	d.Resize(fyne.NewSize(480, 260))
	d.Show()
}
