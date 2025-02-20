package cli

import (
	"os/exec"
	"runtime"
)

func AppLauncher(args ...string) (cmd *exec.Cmd, err error) {
	// Considering the newest app installer that register the ap in the user PATH
	// - on mac os, app and dp2 are registered using a link put in /usr/local/bin by the pkg installer
	// - on windows, the app and dp2 parent folders are directly added to the user PATH env var.
	appExecutable := "DAISY Pipeline"
	switch runtime.GOOS {
	case "windows":
		appExecutable = "DAISY Pipeline.exe"
	}
	path, _err := exec.LookPath(appExecutable)
	if _err == nil {
		cmd = exec.Command(path, args...)
	} else {
		err = _err
	}
	return
}
