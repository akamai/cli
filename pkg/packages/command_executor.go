package packages

import (
	"os"
	"os/exec"
	"runtime"
)

type (
	executor interface {
		// ExecCommand runs the given *exec.Cmd with combined output, if required
		ExecCommand(cmd *exec.Cmd, withCombinedOutput ...bool) ([]byte, error)
		// LookPath searches for an executable named file in the directories named by the PATH environment variable.
		// If file contains a slash, it is tried directly and the PATH is not consulted.
		// The result may be an absolute path or a path relative to the current directory.
		// Just a wrapper around exec.LookPath(string)
		LookPath(string) (string, error)
		// FileExists checks if the given path exists. If there is an error, it will be of type *os.PathError.
		FileExists(string) (bool, error)
		// GetOS is a wrapper around runtime.GOOS
		GetOS() string
	}

	defaultExecutor struct{}
)

func (d *defaultExecutor) ExecCommand(cmd *exec.Cmd, withCombinedOutput ...bool) ([]byte, error) {
	if len(withCombinedOutput) > 0 {
		return cmd.CombinedOutput()
	}
	return cmd.Output()
}

func (d *defaultExecutor) GetOS() string {
	return runtime.GOOS
}

func (d *defaultExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (d *defaultExecutor) FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
