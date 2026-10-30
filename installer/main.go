package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed all:payload
var embedded embed.FS
var version = "1.0.0"

var bundleFiles = []string{"vencordDesktopMain.js", "vencordDesktopPreload.js", "vencordDesktopRenderer.js", "vencordDesktopRenderer.css"}

type savedFile struct {
	name     string
	original []byte
	existed  bool
	value    map[string]any
}

func loadConfig(dir, name string) (savedFile, error) {
	s := savedFile{name: name, value: map[string]any{}}
	b, err := os.ReadFile(filepath.Join(dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	s.original, s.existed = b, true
	// Windows PowerShell 5.1 may have written UTF-8 with BOM.
	if len(b) >= 3 && string(b[:3]) == "\xef\xbb\xbf" {
		b = b[3:]
	}
	if err := json.Unmarshal(b, &s.value); err != nil {
		return s, fmt.Errorf("%s: JSON invalido: %w", name, err)
	}
	if s.value == nil {
		return s, fmt.Errorf("%s: esperado um objeto JSON", name)
	}
	return s, nil
}

func writeAtomic(file string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(file), ".golive-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err := f.Write(content); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

func install(data, root string, payload fs.FS) (err error) {
	required := append([]string{}, bundleFiles...)
	helper := "bin/linux-x64/proton-confgen"
	if runtime.GOOS == "windows" {
		helper = "bin/win32-x64/proton-confgen.exe"
	}
	required = append(required, helper)
	for _, name := range required {
		entry, e := fs.Stat(payload, name)
		if e != nil || !entry.Mode().IsRegular() || entry.Size() == 0 {
			return fmt.Errorf("pacote incompleto: %s", name)
		}
	}
	files := make([]savedFile, 0, 3)
	for _, name := range []string{"state.json", "settings.json", "settings/settings.json"} {
		s, e := loadConfig(data, name)
		if e != nil {
			return e
		}
		files = append(files, s)
	}
	plugins, ok := files[2].value["plugins"].(map[string]any)
	if !ok {
		if files[2].value["plugins"] != nil {
			return errors.New("configuracao plugins invalida; arquivo preservado")
		}
		plugins = map[string]any{}
	}
	plugin, ok := plugins["GoLiveBypass"].(map[string]any)
	if !ok {
		if plugins["GoLiveBypass"] != nil {
			return errors.New("configuracao GoLiveBypass invalida; arquivo preservado")
		}
		plugin = map[string]any{}
	}
	for _, dir := range []string{"builds", "backups"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			return err
		}
	}
	build, err := os.MkdirTemp(filepath.Join(root, "builds"), "v"+version+"-")
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(build)
		}
	}()
	if err := fs.WalkDir(payload, ".", func(name string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if name == "." || d.IsDir() || name == ".gitkeep" {
			return nil
		}
		if !fs.ValidPath(name) || d.Type()&fs.ModeSymlink != 0 {
			return errors.New("caminho invalido no pacote")
		}
		b, e := fs.ReadFile(payload, name)
		if e != nil {
			return e
		}
		p := filepath.Join(build, filepath.FromSlash(name))
		if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return e
		}
		mode := fs.FileMode(0600)
		if name == helper || filepath.Ext(name) == ".exe" {
			mode = 0700
		}
		return os.WriteFile(p, b, mode)
	}); err != nil {
		return err
	}
	backup, err := os.MkdirTemp(filepath.Join(root, "backups"), "install-")
	if err != nil {
		return err
	}
	for _, s := range files {
		if s.existed {
			if err := writeAtomic(filepath.Join(backup, s.name), s.original); err != nil {
				return err
			}
		}
	}
	files[0].value["vencordDir"] = build
	delete(files[1].value, "vencordDir")
	plugin["enabled"], plugin["autoUpdate"], plugin["updateChannel"] = true, false, "stable"
	plugins["GoLiveBypass"] = plugin
	files[2].value["plugins"] = plugins
	// Commit state.json last, after the build and plugin configuration are ready.
	for _, i := range []int{1, 2, 0} {
		b, e := json.MarshalIndent(files[i].value, "", "    ")
		if e == nil {
			e = writeAtomic(filepath.Join(data, files[i].name), append(b, '\n'))
		}
		if e != nil {
			var restoreErrors []error
			for _, s := range files {
				p := filepath.Join(data, s.name)
				if s.existed {
					restoreErrors = append(restoreErrors, writeAtomic(p, s.original))
				} else if e := os.Remove(p); e != nil && !errors.Is(e, os.ErrNotExist) {
					restoreErrors = append(restoreErrors, e)
				}
			}
			return errors.Join(fmt.Errorf("nao foi possivel salvar configuracoes: %w; backup: %s", e, backup), errors.Join(restoreErrors...))
		}
	}
	committed = true
	return nil
}

func run() error {
	dataArg := flag.String("data-dir", "", "Perfil Vesktop personalizado")
	exeArg := flag.String("exe", "", "Executavel Vesktop personalizado")
	noLaunch := flag.Bool("no-launch", false, "Nao abrir o Vesktop depois de instalar")
	showVersion := flag.Bool("version", false, "Mostrar versao")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return nil
	}
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		return errors.New("esta versao atende Windows e Linux x64; macOS ainda nao e suportado")
	}
	if err := checkUser(); err != nil {
		return err
	}
	exe := *exeArg
	var err error
	if exe == "" && !*noLaunch {
		exe, err = findVesktop()
		if err != nil {
			return err
		}
	}
	if exe != "" {
		if _, err := os.Stat(exe); err != nil {
			return fmt.Errorf("Vesktop nao encontrado: %w", err)
		}
	}
	data := *dataArg
	if data == "" {
		data, err = profileDir(exe)
		if err != nil {
			return err
		}
	}
	data, err = filepath.Abs(data)
	if err != nil {
		return err
	}
	root, err := storageDir()
	if err != nil {
		return err
	}
	payload, err := fs.Sub(embedded, "payload")
	if err != nil {
		return err
	}
	fmt.Println("GoLiveBypass para Vesktop —", version)
	fmt.Println("Fechando o Vesktop para preservar as configuracoes...")
	if err := stopVesktop(); err != nil {
		return err
	}
	if err := install(data, root, payload); err != nil {
		if exe != "" && !*noLaunch {
			_ = launchVesktop(exe)
		}
		return err
	}
	if !*noLaunch {
		if err := launchVesktop(exe); err != nil {
			return fmt.Errorf("plugin instalado; nao consegui abrir o Vesktop: %w", err)
		}
	}
	fmt.Println("Instalado e ativado. Conclua o assistente do GoLiveBypass dentro do Vesktop.")
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Falha:", err)
		os.Exit(1)
	}
}
