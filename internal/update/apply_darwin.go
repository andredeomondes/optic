package update

import "os/exec"

// Apply opens the downloaded DMG; the user drags OPTIC to Applications.
// The app keeps running (quit=false).
func Apply(t Target, path, exe string, pid int, dir, version string) (quit bool, err error) {
	if t.Kind != KindDMG {
		return false, ErrUnsupported
	}
	return false, exec.Command("/usr/bin/open", path).Run()
}
