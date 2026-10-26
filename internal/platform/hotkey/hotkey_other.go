//go:build !windows

package hotkey

// Supported reports whether global hotkeys work on this OS.
func Supported() bool { return false }

// Start returns a manager that rejects every binding outside Windows.
func Start(func(id string)) Manager { return noop{} }

type noop struct{}

func (noop) Set(bindings []Binding) []Result { return failAll(bindings, errUnsupported) }
func (noop) Close()                          {}
