package arp

import (
	"os/exec"
	"syscall"
)

// hideWindow keeps a console from flashing when the GUI app runs arp.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
