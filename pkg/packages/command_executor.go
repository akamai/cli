package packages

import (
	"os"
	"os/exec"
)

type (
	executor interface {
		ExecCommand(cmd *exec.Cmd, withCombinedOutput ...bool) ([]byte, error)
		LookPath(string) (string, error)
		FileExists(string) (bool, error)
	}

	defaultExecutor struct{}
)

func (d *defaultExecutor) ExecCommand(cmd *exec.Cmd, withCombinedOutput ...bool) ([]byte, error) {
	if len(withCombinedOutput) > 0 {
		return cmd.CombinedOutput()
	}
	return cmd.Output()
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
