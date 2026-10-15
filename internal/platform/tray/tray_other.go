//go:build !windows

package tray

// Start returns a no-op tray outside Windows.
func Start(Options) Tray { return noop{} }
