package update

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// Apply unpacks the tarball next to the download (<updates>/<v>/OPTIC) and
// opens that folder so the user can run install.sh. The app keeps running
// (quit=false).
func Apply(t Target, path, sha256, exe string, pid int, dir, version string) (quit bool, err error) {
	if t.Kind != KindTarball || !filepath.IsAbs(path) {
		return false, ErrUnsupported
	}
	dest := filepath.Dir(path)
	if err := extractTarball(path, dest, defaultExtractLimits); err != nil {
		return false, err
	}
	folder := filepath.Join(dest, tarRoot)
	xdg, err := exec.LookPath("xdg-open")
	if err != nil {
		return false, fmt.Errorf("%w (%s)", ErrNoXdgOpen, folder)
	}
	cmd := exec.Command(xdg, folder)
	if err := cmd.Start(); err != nil {
		return false, err
	}
	go cmd.Wait() // reap; xdg-open returns quickly
	return false, nil
}
