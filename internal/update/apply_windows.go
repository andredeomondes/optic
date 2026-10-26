package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const (
	createNewProcessGroup = 0x00000200
	detachedProcess       = 0x00000008
)

// Apply writes the update script next to the download (path) and starts it
// detached. exe and pid are the running app; dir is the updates folder that
// receives result.json. It returns quit=true: the app must exit so the
// script can replace it and start it again.
func Apply(t Target, path, exe string, pid int, dir, version string) (quit bool, err error) {
	if t.Kind != KindInstaller && t.Kind != KindPortable {
		return false, ErrUnsupported
	}
	if !validVersion(version) || path == "" || exe == "" {
		return false, ErrUnsupported
	}
	ps := powerShellPath(os.Getenv("SystemRoot"))
	if ps == "" {
		return false, ErrUnsupported
	}
	resultPath := filepath.Join(dir, ResultFile)
	os.Remove(resultPath) // never report a stale outcome
	script := filepath.Join(filepath.Dir(path), scriptName)
	text := windowsScript(t.Kind, path, exe, pid, resultPath, version)
	if err := os.WriteFile(script, scriptBytes(text), 0o600); err != nil {
		return false, err
	}
	cmd := exec.Command(ps, powerShellArgs(script)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNewProcessGroup | detachedProcess,
	}
	if err := cmd.Start(); err != nil {
		return false, err
	}
	cmd.Process.Release()
	return true, nil
}
