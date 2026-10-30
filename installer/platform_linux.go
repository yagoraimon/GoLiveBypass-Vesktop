package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func checkUser() error {
	if os.Geteuid() == 0 {
		return errors.New("execute como seu usuario normal, sem sudo; a autorizacao da VPN acontece depois no plugin")
	}
	return nil
}

func vesktopProcesses() map[int]string {
	result := map[int]string{}
	entries, _ := os.ReadDir("/proc")
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		dir := filepath.Join("/proc", e.Name())
		stat, err := os.Stat(dir)
		if err != nil || stat.Sys().(*syscall.Stat_t).Uid != uint32(os.Getuid()) {
			continue
		}
		exe, err := os.Readlink(filepath.Join(dir, "exe"))
		if err != nil || !strings.EqualFold(filepath.Base(exe), "vesktop") {
			continue
		}
		env, _ := os.ReadFile(filepath.Join(dir, "environ"))
		for _, v := range strings.Split(string(env), "\x00") {
			if strings.HasPrefix(v, "APPIMAGE=") {
				exe = strings.TrimPrefix(v, "APPIMAGE=")
			}
		}
		result[pid] = exe
	}
	return result
}

func findVesktop() (string, error) {
	for _, p := range vesktopProcesses() {
		if strings.HasPrefix(p, "/app/") || strings.HasPrefix(p, "/snap/") || strings.Contains(p, "/flatpak/") {
			continue
		}
		if _, e := os.Stat(p); e == nil {
			return p, nil
		}
	}
	if p, e := exec.LookPath("vesktop"); e == nil {
		real, _ := filepath.EvalSymlinks(p)
		if !strings.Contains(real, "/snap/") && !strings.Contains(real, "/flatpak/") {
			return p, nil
		}
	}
	home, _ := os.UserHomeDir()
	for _, pattern := range []string{"*Vesktop*.AppImage", "*vesktop*.AppImage"} {
		matches, _ := filepath.Glob(filepath.Join(home, "Applications", pattern))
		if len(matches) == 1 {
			return matches[0], nil
		}
	}
	return "", errors.New("Vesktop nativo/AppImage nao encontrado. Flatpak/Snap nao sao suportados pelo motor VPN; abra o AppImage e execute novamente")
}

func profileDir(_ string) (string, error) {
	if p := os.Getenv("VENCORD_USER_DATA_DIR"); p != "" {
		return p, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	for _, name := range []string{"vesktop", "Vesktop"} {
		p := filepath.Join(base, name)
		if _, e := os.Stat(filepath.Join(p, "state.json")); e == nil {
			return p, nil
		}
	}
	return filepath.Join(base, "vesktop"), nil
}

func storageDir() (string, error) {
	p := os.Getenv("XDG_DATA_HOME")
	if p == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(p, "golivebypass-vesktop"), nil
}

func stopVesktop() error {
	for pid := range vesktopProcesses() {
		p, _ := os.FindProcess(pid)
		if err := p.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return err
		}
	}
	for i := 0; i < 50; i++ {
		if len(vesktopProcesses()) == 0 {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return errors.New("o Vesktop continua aberto; feche-o e execute novamente. Configuracoes preservadas")
}

func launchVesktop(exe string) error {
	cmd := exec.Command(exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
