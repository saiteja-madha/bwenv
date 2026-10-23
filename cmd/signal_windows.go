//go:build windows

package cmd

import "os/exec"

func signaledExitCode(*exec.ExitError) (int, bool) {
	return 0, false
}
