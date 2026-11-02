package update

import (
	"os/exec"
	"path/filepath"
	"time"
)

// Apply opens the downloaded DMG; the user drags OPTIC to Applications.
// The app keeps running (quit=false).
func Apply(t Target, path, sha256, exe string, pid int, dir, version string) (quit bool, err error) {
	if t.Kind != KindDMG || !filepath.IsAbs(path) {
		return false, ErrUnsupported
	}
	// Quarantine it like a browser download so Gatekeeper checks the DMG.
	// Best effort: a failure here must not stop the update.
	exec.Command("/usr/bin/xattr", "-w", "com.apple.quarantine", macQuarantine(time.Now()), path).Run()
	return false, exec.Command("/usr/bin/open", path).Run()
}
