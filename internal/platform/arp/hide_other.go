//go:build !windows

package arp

import "os/exec"

func hideWindow(*exec.Cmd) {}
