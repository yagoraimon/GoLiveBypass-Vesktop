package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func bundle() fs.FS {
	p := fstest.MapFS{}
	for _, name := range []string{"vencordDesktopMain.js", "vencordDesktopPreload.js", "vencordDesktopRenderer.js", "vencordDesktopRenderer.css", "bin/win32-x64/proton-confgen.exe", "bin/linux-x64/proton-confgen"} {
		p[name] = &fstest.MapFile{Data: []byte("GoLiveBypass")}
	}
	return p
}

func put(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, name string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestInstallPreservesSettingsAndLoadsBuildFromState(t *testing.T) {
	data, root := t.TempDir(), t.TempDir()
	put(t, data, "state.json", `{"vencordDir":"old-custom-build","windowBounds":{"width":1440}}`)
	put(t, data, "settings.json", `{"vencordDir":"wrong-place","arRPC":true}`)
	put(t, data, "settings/settings.json", `{"customCSS":"preserve","plugins":{"OtherPlugin":{"enabled":true},"GoLiveBypass":{"protonCountry":"NL","onboardingCompleted":true}}}`)
	if err := install(data, root, bundle()); err != nil {
		t.Fatal(err)
	}
	state := read(t, data, "state.json")
	build := state["vencordDir"].(string)
	if _, err := os.Stat(filepath.Join(build, "vencordDesktopMain.js")); err != nil {
		t.Fatal(err)
	}
	if state["windowBounds"].(map[string]any)["width"] != float64(1440) {
		t.Fatal("window state lost")
	}
	appSettings := read(t, data, "settings.json")
	if appSettings["arRPC"] != true || appSettings["vencordDir"] != nil {
		t.Fatal("Vesktop settings not preserved/corrected")
	}
	config := read(t, data, "settings/settings.json")
	if config["customCSS"] != "preserve" {
		t.Fatal("other settings lost")
	}
	plugins := config["plugins"].(map[string]any)
	if plugins["OtherPlugin"].(map[string]any)["enabled"] != true {
		t.Fatal("other plugin settings lost")
	}
	glb := plugins["GoLiveBypass"].(map[string]any)
	if glb["enabled"] != true || glb["autoUpdate"] != false || glb["protonCountry"] != "NL" || glb["onboardingCompleted"] != true {
		t.Fatalf("wrong plugin settings: %v", glb)
	}
	backups, err := filepath.Glob(filepath.Join(root, "backups", "*", "state.json"))
	if err != nil || len(backups) != 1 {
		t.Fatal("original state backup missing")
	}
	if read(t, filepath.Dir(backups[0]), "state.json")["vencordDir"] != "old-custom-build" {
		t.Fatal("backup lost original build")
	}
	if err := install(data, root, bundle()); err != nil {
		t.Fatal("reinstall:", err)
	}
	if read(t, data, "settings/settings.json")["customCSS"] != "preserve" {
		t.Fatal("reinstall lost settings")
	}
}

func TestInstallFirstUseEnablesPluginWithoutCompletingSetup(t *testing.T) {
	data, root := t.TempDir(), t.TempDir()
	if err := install(data, root, bundle()); err != nil {
		t.Fatal(err)
	}
	glb := read(t, data, "settings/settings.json")["plugins"].(map[string]any)["GoLiveBypass"].(map[string]any)
	if glb["enabled"] != true || glb["onboardingCompleted"] != nil {
		t.Fatal("setup must remain pending")
	}
}

func TestInstallRejectsInvalidConfigurationWithoutChanges(t *testing.T) {
	for _, name := range []string{"state.json", "settings.json", "settings/settings.json"} {
		t.Run(name, func(t *testing.T) {
			data, root := t.TempDir(), t.TempDir()
			for _, file := range []string{"state.json", "settings.json", "settings/settings.json"} {
				put(t, data, file, "{}")
			}
			put(t, data, name, "not json")
			if err := install(data, root, bundle()); err == nil {
				t.Fatal("invalid JSON accepted")
			}
			for _, file := range []string{"state.json", "settings.json", "settings/settings.json"} {
				b, _ := os.ReadFile(filepath.Join(data, file))
				want := "{}"
				if file == name {
					want = "not json"
				}
				if string(b) != want {
					t.Fatal("config changed on failure:", file)
				}
			}
		})
	}
}

func TestIncompleteBundleLeavesOriginalBuild(t *testing.T) {
	data, root := t.TempDir(), t.TempDir()
	put(t, data, "state.json", `{"vencordDir":"original"}`)
	if err := install(data, root, fstest.MapFS{}); err == nil {
		t.Fatal("incomplete bundle accepted")
	}
	if read(t, data, "state.json")["vencordDir"] != "original" {
		t.Fatal("original build lost")
	}
}
