//go:build !windows && !darwin && !linux

package update

// Apply is not available on this system.
func Apply(t Target, path, sha256, exe string, pid int, dir, version string) (quit bool, err error) {
	return false, ErrUnsupported
}
