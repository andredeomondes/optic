package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	createNewProcessGroup = 0x00000200
	detachedProcess       = 0x00000008
)

// Apply writes the update script next to the download (path, whose verified
// SHA-256 is sha256) and starts it detached. exe and pid are the running
// app; dir is the updates folder that receives result.json. It returns
// quit=true only once the script has proven it runs; if Windows refuses it,
// the app stays open and ErrApplyBlocked is returned.
func Apply(t Target, path, sha256, exe string, pid int, dir, version string) (quit bool, err error) {
	if err := validateApply(t, path, sha256, exe, pid, dir, version); err != nil {
		return false, err
	}
	ps := powerShellPath(windowsDir())
	if ps == "" {
		return false, ErrUnsupported
	}
	work := filepath.Dir(path)
	resultPath := filepath.Join(dir, ResultFile)
	marker := filepath.Join(work, markerName)
	cancel := filepath.Join(work, cancelName)
	os.Remove(resultPath) // never report a stale outcome
	if err := clearStale(marker, cancel); err != nil {
		return false, err
	}

	// Best effort: mark the download as from the Internet so SmartScreen
	// checks it like a browser download.
	os.WriteFile(path+":Zone.Identifier", []byte(zoneIdentifier), 0o644)

	script := filepath.Join(work, scriptName)
	text := windowsScript(t.Kind, path, sha256, exe, pid, resultPath, marker, cancel, version)
	// On Windows the mode only decides the read-only bit.
	if err := os.WriteFile(script, scriptBytes(text), 0o600); err != nil {
		return false, fmt.Errorf("%w: %v", ErrDest, err)
	}
	cmd := exec.Command(ps, powerShellArgs(script)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNewProcessGroup | detachedProcess,
	}
	if err := cmd.Start(); err != nil {
		return false, ErrApplyBlocked
	}
	exited := make(chan struct{})
	go func() { cmd.Wait(); close(exited) }()
	markerExists := func() bool { _, err := os.Stat(marker); return err == nil }
	if err := awaitStart(markerExists, exited, applyStartTimeout, applyStartPoll); err != nil {
		// A script that starts after this sees the cancel file and exits.
		cancelStart(cancel)
		return false, err
	}
	return true, nil
}

// windowsDir is the Windows folder from GetSystemWindowsDirectoryW, falling
// back to %SystemRoot%.
func windowsDir() string {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemWindowsDirectoryW")
	if proc.Find() == nil {
		buf := make([]uint16, syscall.MAX_PATH)
		n, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n > 0 && int(n) < len(buf) {
			return syscall.UTF16ToString(buf[:n])
		}
	}
	return os.Getenv("SystemRoot")
}
