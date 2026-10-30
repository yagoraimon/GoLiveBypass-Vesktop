# GoLiveBypass-Vesktop

Only the GoLiveBypass plugin, Proton helper and Vesktop installers belong here. Windows/Linux x64; macOS is explicitly deferred.

Preserve user settings, source attribution and network isolation. Never replace Vesktop app.asar. vencordDir belongs in Vesktop state.json. Source-based upstream updates must stay disabled for the prebuilt distribution. Installation must not require development tools or administrative privileges.

Run `go test ./...` in installer and the CI Windows/Linux tests. Read tests before running anything that operates a real Vesktop or VPN. Do not publish unrelated upstream apps or run inherited upstream workflows. Repository: yagoraimon/GoLiveBypass-Vesktop.
