// Package update picks the release asset that fits this install, downloads it
// from GitHub and checks it against the release's SHA256SUMS. It never runs
// anything it downloads.
package update

import (
	"errors"
	pathpkg "path"
	"strings"
)

// Kind is how OPTIC is installed, which decides the asset and how to apply it.
type Kind string

const (
	KindInstaller Kind = "installer" // Windows, installed under Program Files
	KindPortable  Kind = "portable"  // Windows, single exe anywhere else
	KindDMG       Kind = "dmg"       // macOS
	KindTarball   Kind = "tarball"   // Linux
)

// Release asset names, as uploaded by the release workflow.
const (
	AssetWindowsSetup    = "OPTIC-windows-amd64-setup.exe"
	AssetWindowsPortable = "OPTIC-windows-amd64-portable.exe"
	AssetMacOS           = "OPTIC-macos-universal.dmg"
	AssetLinux           = "OPTIC-linux-amd64.tar.gz"
	AssetLinuxWebkit41   = "OPTIC-linux-amd64-webkit41.tar.gz"
	AssetSums            = "SHA256SUMS"
)

// Target is the asset to download and the kind of install it updates.
type Target struct {
	Kind  Kind
	Asset string
}

// Errors shown to the user (the UI translates them).
var (
	ErrNoChecksum  = errors.New("Esta versão não tem lista de checksums; atualize pela página de releases.")
	ErrNoAsset     = errors.New("Esta versão não tem um arquivo para este sistema.")
	ErrChecksum    = errors.New("O download não confere (checksum). Tente de novo.")
	ErrTooLarge    = errors.New("Download maior que o esperado; cancelado.")
	ErrNetwork     = errors.New("Não foi possível baixar a atualização. Verifique a conexão e tente de novo.")
	ErrUnsupported = errors.New("Atualização automática não disponível neste sistema.")
)

// Choose picks the asset for the running install. exePath is the running
// executable as os.Executable returns it (a full long path, not an 8.3
// short name); programFiles are %ProgramFiles% and %ProgramFiles(x86)%
// (empty entries are ignored); webkit41 is true for builds tagged
// webkit2_41. Only amd64 builds exist for Windows and Linux; the macOS dmg
// is universal.
func Choose(goos, goarch, exePath string, programFiles []string, webkit41 bool) (Target, error) {
	switch {
	case goos == "windows" && goarch == "amd64":
		if underAny(exePath, programFiles) {
			return Target{KindInstaller, AssetWindowsSetup}, nil
		}
		return Target{KindPortable, AssetWindowsPortable}, nil
	case goos == "darwin":
		return Target{KindDMG, AssetMacOS}, nil
	case goos == "linux" && goarch == "amd64":
		if webkit41 {
			return Target{KindTarball, AssetLinuxWebkit41}, nil
		}
		return Target{KindTarball, AssetLinux}, nil
	}
	return Target{}, ErrUnsupported
}

// underAny reports whether a Windows path lies inside one of dirs, ignoring
// case and slash style. Paths are cleaned first so ".." cannot fake it; the
// cleaning is done by hand so the result does not depend on the host OS.
func underAny(path string, dirs []string) bool {
	norm := func(s string) string {
		s = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), `\`, "/"))
		if s == "" {
			return ""
		}
		return pathpkg.Clean(s)
	}
	p := norm(path)
	for _, d := range dirs {
		d = strings.TrimRight(norm(d), "/")
		if d != "" && strings.HasPrefix(p, d+"/") {
			return true
		}
	}
	return false
}

// knownAsset reports whether name is one of the installable assets.
func knownAsset(name string) bool {
	switch name {
	case AssetWindowsSetup, AssetWindowsPortable, AssetMacOS, AssetLinux, AssetLinuxWebkit41:
		return true
	}
	return false
}
