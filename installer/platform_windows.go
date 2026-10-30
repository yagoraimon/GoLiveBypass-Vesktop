package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func checkUser() error { return nil }

func findVesktop() (string, error) {
	b, _ := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Get-Process -Name vesktop -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Path -Unique").Output()
	candidates := strings.Split(strings.TrimSpace(string(b)), "\n")
	for _, base := range []string{os.Getenv("LOCALAPPDATA"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
		if base != "" {
			candidates = append(candidates, filepath.Join(base, "vesktop", "vesktop.exe"))
		}
	}
	for _, p := range candidates {
		p = strings.TrimSpace(p)
		if p != "" {
			if s, e := os.Stat(p); e == nil && !s.IsDir() {
				return p, nil
			}
		}
	}
	return "", errors.New("Vesktop nao encontrado. Instale o Vesktop ou use --exe para indicar uma instalacao portable")
}

func profileDir(exe string) (string, error) {
	if p := os.Getenv("VENCORD_USER_DATA_DIR"); p != "" {
		return p, nil
	}
	if exe != "" {
		if _, err := os.Stat(filepath.Join(filepath.Dir(exe), "Uninstall Vesktop.exe")); errors.Is(err, os.ErrNotExist) {
			return filepath.Join(filepath.Dir(exe), "Data"), nil
		}
	}
	if p := os.Getenv("APPDATA"); p != "" {
		return filepath.Join(p, "Vesktop"), nil
	}
	return "", errors.New("APPDATA nao definido; perfil Vesktop nao encontrado")
}

func storageDir() (string, error) {
	p, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(p, "GoLiveBypass-Vesktop"), nil
}

func vesktopRunning() bool {
	b, err := exec.Command("tasklist.exe", "/FI", "IMAGENAME eq vesktop.exe", "/FO", "CSV", "/NH").Output()
	return err == nil && strings.Contains(strings.ToLower(string(b)), "\"vesktop.exe\"")
}

func stopVesktop() error {
	if !vesktopRunning() {
		return nil
	}
	if b, err := exec.Command("taskkill.exe", "/IM", "vesktop.exe", "/T", "/F").CombinedOutput(); err != nil {
		return fmt.Errorf("nao consegui fechar o Vesktop: %s", strings.TrimSpace(string(b)))
	}
	for i := 0; i < 30; i++ {
		if !vesktopRunning() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return errors.New("o Vesktop continua aberto; configuracoes preservadas")
}

func launchVesktop(exe string) error {
	cmd := exec.Command(exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
