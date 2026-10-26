// Package update picks the release asset that fits this install, downloads it
// from GitHub and checks it against the release's SHA256SUMS. It never runs
// anything it downloads.
package update

import (
	"errors"
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
// executable; programFiles are %ProgramFiles% and %ProgramFiles(x86)% (empty
// entries are ignored); webkit41 is true for builds tagged webkit2_41.
func Choose(goos, exePath string, programFiles []string, webkit41 bool) (Target, error) {
	switch goos {
	case "windows":
		if underAny(exePath, programFiles) {
			return Target{KindInstaller, AssetWindowsSetup}, nil
		}
		return Target{KindPortable, AssetWindowsPortable}, nil
	case "darwin":
		return Target{KindDMG, AssetMacOS}, nil
	case "linux":
		if webkit41 {
			return Target{KindTarball, AssetLinuxWebkit41}, nil
		}
		return Target{KindTarball, AssetLinux}, nil
	}
	return Target{}, ErrUnsupported
}

// underAny reports whether a Windows path lies inside one of dirs,
// ignoring case and slash style.
func underAny(path string, dirs []string) bool {
	norm := func(s string) string {
		return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "/", `\`))
	}
	p := norm(path)
	for _, d := range dirs {
		d = strings.TrimRight(norm(d), `\`)
		if d != "" && strings.HasPrefix(p, d+`\`) {
			return true
		}
	}
	return false
}
