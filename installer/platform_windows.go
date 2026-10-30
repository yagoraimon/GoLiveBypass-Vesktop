package main

import (
	"encoding/csv"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

func checkUser() error { return nil }

func findVesktop() (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "[Console]::OutputEncoding=[Text.UTF8Encoding]::new($false); Get-Process -Name vesktop -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Path -Unique")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	b, _ := cmd.Output()
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

func vesktopPIDs() (map[uint32]bool, error) {
	cmd := exec.Command("tasklist.exe", "/FI", "IMAGENAME eq vesktop.exe", "/FO", "CSV", "/NH")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	b, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	rows, err := csv.NewReader(strings.NewReader(string(b))).ReadAll()
	if err != nil {
		return nil, err
	}
	pids := map[uint32]bool{}
	for _, row := range rows {
		if len(row) > 1 && strings.EqualFold(row[0], "vesktop.exe") {
			pid, err := strconv.ParseUint(row[1], 10, 32)
			if err == nil {
				pids[uint32(pid)] = true
			}
		}
	}
	return pids, nil
}

func vesktopRunning() bool {
	pids, err := vesktopPIDs()
	return err != nil || len(pids) > 0
}

// Invoke Vesktop's native Quit menu item, which calls Electron app.quit().
func requestMenuQuit() {
	pids, err := vesktopPIDs()
	if err != nil {
		return
	}
	user32 := syscall.NewLazyDLL("user32.dll")
	getMenu := user32.NewProc("GetMenu")
	count := user32.NewProc("GetMenuItemCount")
	subMenu := user32.NewProc("GetSubMenu")
	menuString := user32.NewProc("GetMenuStringW")
	menuID := user32.NewProc("GetMenuItemID")
	windowPID := user32.NewProc("GetWindowThreadProcessId")
	post := user32.NewProc("PostMessageW")
	className := user32.NewProc("GetClassNameW")
	owner := user32.NewProc("GetWindow")
	show := user32.NewProc("ShowWindow")
	foreground := user32.NewProc("GetForegroundWindow")
	activate := user32.NewProc("SetForegroundWindow")
	keyboard := user32.NewProc("keybd_event")
	var quitID func(uintptr, int) uintptr
	quitID = func(menu uintptr, depth int) uintptr {
		if menu == 0 || depth > 8 {
			return 0
		}
		n, _, _ := count.Call(menu)
		if int32(n) < 0 || n > 128 {
			return 0
		}
		for i := uintptr(0); i < n; i++ {
			var text [256]uint16
			menuString.Call(menu, i, uintptr(unsafe.Pointer(&text[0])), uintptr(len(text)), 0x400)
			label := strings.TrimSpace(strings.ReplaceAll(strings.SplitN(syscall.UTF16ToString(text[:]), "\t", 2)[0], "&", ""))
			if strings.EqualFold(label, "Quit") {
				id, _, _ := menuID.Call(menu, i)
				if id != 0 && uint32(id) != ^uint32(0) {
					return id
				}
			}
			child, _, _ := subMenu.Call(menu, i)
			if id := quitID(child, depth+1); id != 0 {
				return id
			}
		}
		return 0
	}
	callback := syscall.NewCallback(func(window, _ uintptr) uintptr {
		var pid uint32
		windowPID.Call(window, uintptr(unsafe.Pointer(&pid)))
		parent, _, _ := owner.Call(window, 4)
		var class [128]uint16
		className.Call(window, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
		if pids[pid] && parent == 0 && syscall.UTF16ToString(class[:]) == "Chrome_WidgetWin_1" {
			menu, _, _ := getMenu.Call(window)
			if id := quitID(menu, 0); id != 0 {
				post.Call(window, 0x111, id, 0)
				return 0
			}
			// Chromium may render its own menu. Vesktop registers Alt+F4 as app.quit().
			show.Call(window, 9)
			activate.Call(window)
			current, _, _ := foreground.Call()
			if current == window {
				keyboard.Call(0x12, 0x38, 0, 0)
				keyboard.Call(0x73, 0x3e, 0, 0)
				keyboard.Call(0x73, 0x3e, 2, 0)
				keyboard.Call(0x12, 0x38, 2, 0)
				return 0
			}
		}
		return 1
	})
	user32.NewProc("EnumWindows").Call(callback, 0)
}

func stopVesktop(data string) error {
	if !vesktopRunning() {
		return nil
	}
	defer os.Remove(filepath.Join(data, shutdownFile))
	accepted, err := requestPackagedExit(data)
	if err != nil {
		return err
	}
	if !accepted {
		requestMenuQuit()
	}
	if waitForVesktopExit() {
		return nil
	}
	return errors.New("o Vesktop nao aceitou o encerramento automatico. Use Sair/Quit no menu do Vesktop e execute novamente; nenhuma configuracao foi alterada")
}

func launchVesktop(exe, data string) error {
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), "VENCORD_USER_DATA_DIR="+data)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
