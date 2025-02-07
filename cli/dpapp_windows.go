// build: windows
package cli

import (
	"os"
	"os/exec"
	"path/filepath"
)

func appPathWindows() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "pipeline-ui", "DAISY Pipeline.exe")
}

func AppLauncher(args ...string) (cmd *exec.Cmd, err error) {
	if _, _err := os.Stat(appPathWindows()); _err == nil {
		cmd = exec.Command(appPathWindows(), args...)
	} else {
		err = _err
	}
	return
}
