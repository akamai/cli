// Copyright 2018. Akamai Technologies, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package packages

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/tools"
	"github.com/akamai/cli/v2/pkg/version"
)

var (
	pythonVersionPattern = `Python ([2,3]\.\d+\.\d+).*`
	pythonVersionRegex   = regexp.MustCompile(pythonVersionPattern)
	pipVersionPattern    = `^pip \d{1,2}\..+ \(python [2,3]\.\d+\)`
	venvHelpPattern      = `usage: venv `
	pipVersionRegex      = regexp.MustCompile(pipVersionPattern)
	venvHelpRegex        = regexp.MustCompile(venvHelpPattern)
)

func (l *langManager) installPython(ctx context.Context, venvPath, srcPath, requiredPy string) error {
	logger := log.FromContext(ctx)

	pythonBin, pipBin, err := l.validatePythonDeps(ctx, logger, requiredPy, filepath.Base(srcPath))
	if err != nil {
		logger.Error(fmt.Sprintf("%v", err))
		return err
	}

	if err = l.setup(ctx, venvPath, srcPath, pythonBin, pipBin, requiredPy, false); err != nil {
		logger.Error(fmt.Sprintf("%v", err))
		return err
	}

	return nil
}

// setup does the python virtualenv set up for the given module. It may return an error.
func (l *langManager) setup(ctx context.Context, pkgVenvPath, srcPath, python3Bin, pipBin, requiredPy string, passthru bool) error {
	switch version.Compare(requiredPy, "3.0.0") {
	case version.Greater, version.Equals:
		// Python 3.x required: build virtual environment
		logger := log.FromContext(ctx)

		defer func() {
			if !passthru {
				l.deactivateVirtualEnvironment(ctx, pkgVenvPath, requiredPy)
			}
			logger.Debug("All virtualenv dependencies successfully installed")
		}()

		veExists, err := l.commandExecutor.FileExists(pkgVenvPath)
		if err != nil {
			return err
		}

		if !passthru || !veExists {
			logger.Debug(fmt.Sprintf("the virtual environment %s does not exist yet - installing dependencies", pkgVenvPath))

			// upgrade pip and setuptools
			if err := l.upgradePipAndSetuptools(ctx, python3Bin); err != nil {
				return err
			}

			// create virtual environment
			if err := l.createVirtualEnvironment(ctx, python3Bin, pkgVenvPath); err != nil {
				return err
			}
		}

		// activate virtual environment
		if err := l.activateVirtualEnvironment(ctx, pkgVenvPath); err != nil {
			return err
		}

		// install packages from requirements.txt
		vePy, err := l.getVePython(pkgVenvPath)
		if err != nil {
			return err
		}
		if err := l.installVeRequirements(ctx, srcPath, pkgVenvPath, vePy); err != nil {
			return err
		}
	case version.Smaller:
		// no virtualenv for python 2.x
		return installPythonDepsPip(ctx, l.commandExecutor, pipBin, srcPath)
	}

	return nil
}

func (l *langManager) getVePython(vePath string) (string, error) {
	switch l.GetOS() {
	case "windows":
		return filepath.Join(vePath, "System", "python.exe"), nil
	case "linux", "darwin":
		return filepath.Join(vePath, "bin", "python"), nil
	default:
		return "", ErrOSNotSupported
	}
}

func (l *langManager) installVeRequirements(ctx context.Context, srcPath, vePath, py3Bin string) error {
	logger := log.FromContext(ctx)

	requirementsPath := filepath.Join(srcPath, "requirements.txt")
	if ok, _ := l.commandExecutor.FileExists(requirementsPath); !ok {
		return ErrRequirementsTxtNotFound
	}
	logger.Info("requirements.txt found, running pip package manager")

	shell, err := l.GetShell(l.GetOS())
	if err != nil {
		return err
	}
	if shell == "" {
		// windows
		pipPath := filepath.Join(vePath, "Scripts", "pip.exe")
		if output, err := l.commandExecutor.ExecCommand(&exec.Cmd{
			Path: pipPath,
			Args: []string{pipPath, "install", "--upgrade", "--ignore-installed", "-r", requirementsPath},
		}, true); err != nil {
			term := terminal.Get(ctx)
			_, _ = term.Writeln(string(output))
			logger.Error("failed to run pip install --upgrade --ignore-installed -r requirements.txt")
			return fmt.Errorf("%w: %s", ErrRequirementsInstall, string(output))
		}
		return nil
	}

	if output, err := l.commandExecutor.ExecCommand(&exec.Cmd{
		Path: py3Bin,
		Args: []string{py3Bin, "-m", "pip", "install", "--upgrade", "--ignore-installed", "-r", requirementsPath},
	}, true); err != nil {
		logger.Error("failed to run pip install --upgrade --ignore-installed -r requirements.txt")
		logger.Error(string(output))
		return fmt.Errorf("%w: %v", ErrRequirementsInstall, string(output))
	}

	return nil
}

/*
validatePythonDeps does system dependencies validation based on the required python version

It returns:

* route to required python executable

* route to pip executable, for modules which require python < v3

* error, if any
*/
func (l *langManager) validatePythonDeps(ctx context.Context, logger *slog.Logger, requiredPy, name string) (string, string, error) {
	switch version.Compare(requiredPy, "3.0.0") {
	case version.Smaller:
		// v2 required -> no virtualenv
		logger.Debug("Validating dependencies for python 2.x module")
		pythonBin, err := findPythonBin(ctx, l.commandExecutor, requiredPy, "")
		if err != nil {
			logger.Error("Python >= 2 (and < 3.0) not found in the system. Please verify your setup")
			return "", "", err
		}

		if err := l.resolveBinVersion(pythonBin, requiredPy, "--version", logger); err != nil {
			return "", "", err
		}

		pipBin, err := findPipBin(ctx, l.commandExecutor, requiredPy)
		if err != nil {
			return pythonBin, "", err
		}
		return pythonBin, pipBin, nil
	case version.Greater, version.Equals:
		// v3 required -> virtualenv
		// requirements for setting up VE: python3, pip3, venv
		logger.Debug(fmt.Sprintf("Validating dependencies for python %s module", requiredPy))
		pythonBin, err := findPythonBin(ctx, l.commandExecutor, requiredPy, name)
		if err != nil {
			logger.Error(fmt.Sprintf("Python >= %s not found in the system. Please verify your setup", requiredPy))
			return "", "", err
		}

		if err := l.resolveBinVersion(pythonBin, requiredPy, "--version", logger); err != nil {
			return "", "", err
		}

		// validate that the use has python pip package installed
		if err = l.findPipPackage(ctx, requiredPy, pythonBin); err != nil {
			logger.Error("Pip not found in the system. Please verify your setup")
			return "", "", err
		}

		// validate that venv module is present
		if err = l.findVenvPackage(ctx, pythonBin); err != nil {
			logger.Error("Python venv module not found in the system. Please verify your setup")
			return "", "", err
		}

		return pythonBin, "", nil
	default:
		// not supported
		logger.Error(fmt.Sprintf("%s: %s", ErrPythonVersionNotSupported.Error(), requiredPy))
		return "", "", fmt.Errorf("%w: %s", ErrPythonVersionNotSupported, requiredPy)
	}
}

func (l *langManager) findVenvPackage(ctx context.Context, pythonBin string) error {
	logger := log.FromContext(ctx)
	cmd := exec.Command(pythonBin, "-m", "venv", "--version")
	output, _ := l.commandExecutor.ExecCommand(cmd, true)
	logger.Debug(fmt.Sprintf("%s %s: %s", pythonBin, "-m venv --version", bytes.ReplaceAll(output, []byte("\n"), []byte(""))))
	matches := venvHelpRegex.FindStringSubmatch(string(output))
	if len(matches) == 0 {
		return fmt.Errorf("%w: %s", ErrVenvNotFound, bytes.ReplaceAll(output, []byte("\n"), []byte("")))
	}
	return nil
}

func (l *langManager) findPipPackage(ctx context.Context, requiredPy string, pythonBin string) error {
	compare := version.Compare(requiredPy, "3.0.0")
	if compare == version.Greater || compare == version.Equals {
		logger := log.FromContext(ctx)

		// find pip python package, not pip executable
		cmd := exec.Command(pythonBin, "-m", "pip", "--version")
		output, _ := l.commandExecutor.ExecCommand(cmd, true)
		logger.Debug(fmt.Sprintf("%s %s: %s", pythonBin, "-m pip --version", bytes.ReplaceAll(output, []byte("\n"), []byte(""))))
		matches := pipVersionRegex.FindStringSubmatch(string(output))
		if len(matches) == 0 {
			return fmt.Errorf("%w: %s", ErrPipNotFound, bytes.ReplaceAll(output, []byte("\n"), []byte("")))
		}
	}

	return nil
}

func (l *langManager) activateVirtualEnvironment(ctx context.Context, pkgVenvPath string) error {
	logger := log.FromContext(ctx)
	logger.Debug(fmt.Sprintf("Activating Python virtualenv: %s", pkgVenvPath))
	oS := l.GetOS()
	interpreter, err := l.GetShell(oS)
	if err != nil {
		logger.Error("cannot determine OS shell")
		return err
	}
	cmd := &exec.Cmd{}
	if oS == "windows" {
		activate := filepath.Join(pkgVenvPath, "Scripts", "activate.bat")
		cmd.Path = activate
		cmd.Args = []string{}
	} else {
		cmd.Path = interpreter
		cmd.Args = []string{"source", filepath.Join(pkgVenvPath, "bin", "activate")}
	}
	if output, err := l.commandExecutor.ExecCommand(cmd, true); err != nil {
		logger.Error(fmt.Sprintf("%v: %v", ErrVirtualEnvActivation, string(output)))
		return fmt.Errorf("%w: %s", ErrVirtualEnvActivation, string(output))
	}
	logger.Debug(fmt.Sprintf("Python virtualenv %s active", pkgVenvPath))

	return nil
}

func (l *langManager) deactivateVirtualEnvironment(ctx context.Context, dir, pyVersion string) {

	compare := version.Compare(pyVersion, "3.0.0")
	if compare == version.Equals || compare == version.Greater {
		logger := log.FromContext(ctx)
		logger.Debug(fmt.Sprintf("Deactivating virtual environment %s", dir))
		cmd := &exec.Cmd{}
		oS := l.GetOS()
		if oS == "windows" {
			logger.Debug("windows detected, executing deactivate.bat")
			deactivate := filepath.Join(dir, "Scripts", "deactivate.bat")
			cmd.Path = deactivate
			cmd.Args = []string{deactivate}
		} else {
			cmd.Path, _ = l.GetShell(oS)
			cmd.Args = []string{"deactivate"}
		}
		// errors are ignored at this step, as we may be trying to deactivate a non existent or not active virtual environment
		if _, err := l.commandExecutor.ExecCommand(cmd, true); err == nil {
			logger.Debug("Python virtualenv deactivated")
		} else {
			logger.Debug(fmt.Sprintf("Error deactivating VE %s: %v", dir, err))
		}
	}
}

func (l *langManager) resolveBinVersion(bin, cmdReq, arg string, logger *slog.Logger) error {
	cmd := exec.Command(bin, arg)
	output, err := l.commandExecutor.ExecCommand(cmd, true)
	if err != nil {
		return err
	}
	logger.Debug(fmt.Sprintf("%s %s: %s", bin, arg, bytes.ReplaceAll(output, []byte("\n"), []byte(""))))
	matches := pythonVersionRegex.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return fmt.Errorf("%w: %s: %s", ErrRuntimeNoVersionFound, "python", cmd)
	}
	switch version.Compare(cmdReq, matches[1]) {
	case version.Greater:
		// python required > python installed
		logger.Error(fmt.Sprintf("%s version found: %s", bin, matches[1]))
		return fmt.Errorf("%w: required: %s:%s, have: %s. Please install the required Python branch", ErrRuntimeMinimumVersionRequired, bin, cmdReq, matches[1])
	case version.Smaller:
		// python required < python installed
		if version.Compare(cmdReq, "3.0.0") == 1 && version.Compare(matches[1], "3.0.0") <= 0 {
			// required: py2; found: py3. The user still needs to install py2
			logger.Error(fmt.Sprintf("Python version %s found, %s version required. Please, install the %s Python branch.", matches[1], cmdReq, cmdReq))
			return fmt.Errorf("%w: Please install the following Python branch: %s", ErrRuntimeNotFound, cmdReq)
		}
		return nil
	case version.Equals:
		return nil
	}
	return ErrPythonVersionNotSupported
}

func (l *langManager) upgradePipAndSetuptools(ctx context.Context, python3Bin string) error {
	logger := log.FromContext(ctx)

	// if python3 > v3.4, ensure pip
	if err := l.resolveBinVersion(python3Bin, "3.4.0", "--version", logger); err != nil && errors.Is(err, ErrRuntimeNotFound) {
		return err
	}

	logger.Debug("Installing/upgrading pip")

	// ensure pip is present
	cmdPip := exec.Command(python3Bin, "-m", "ensurepip", "--upgrade")
	if output, err := l.commandExecutor.ExecCommand(cmdPip, true); err != nil {
		logger.Warn(fmt.Sprintf("%v: %s", ErrPipUpgrade, string(output)))
	}

	// upgrade pip & setuptools
	logger.Debug("Installing/upgrading pip & setuptools")
	cmdSetuptools := exec.Command(python3Bin, "-m", "pip", "install" /*, "--user"*/, "--no-cache", "--upgrade", "pip", "setuptools")
	if output, err := l.commandExecutor.ExecCommand(cmdSetuptools, true); err != nil {
		logger.Error(fmt.Sprintf("%v: %s", ErrPipSetuptoolsUpgrade, string(output)))
		return fmt.Errorf("%w: %s", ErrPipSetuptoolsUpgrade, string(output))
	}

	return nil
}

func (l *langManager) createVirtualEnvironment(ctx context.Context, python3Bin string, pkgVenvPath string) error {
	logger := log.FromContext(ctx)

	// check if the .akamai-cli/venv directory exists - create it otherwise
	venvPath := filepath.Dir(pkgVenvPath)
	if exists, err := l.commandExecutor.FileExists(venvPath); err == nil && !exists {
		logger.Debug(fmt.Sprintf("%s does not exist; let's create it", venvPath))
		if err := os.Mkdir(venvPath, 0755); err != nil {
			logger.Error(fmt.Sprintf("%v %s: %v", ErrDirectoryCreation, venvPath, err))
			return fmt.Errorf("%w %s: %v", ErrDirectoryCreation, venvPath, err)
		}
		logger.Debug(fmt.Sprintf("%s directory created", venvPath))
	} else {
		if err != nil {
			return err
		}
	}

	logger.Debug(fmt.Sprintf("Creating python virtualenv: %s", pkgVenvPath))
	cmdVenv := exec.Command(python3Bin, "-m", "venv", pkgVenvPath)
	if output, err := l.commandExecutor.ExecCommand(cmdVenv, true); err != nil {
		logger.Error(fmt.Sprintf("%v %s: %s", ErrVirtualEnvCreation, pkgVenvPath, string(output)))
		return fmt.Errorf("%w %s: %s", ErrVirtualEnvCreation, pkgVenvPath, string(output))
	}
	logger.Debug(fmt.Sprintf("Python virtualenv successfully created: %s", pkgVenvPath))

	return nil
}

func findPythonBin(ctx context.Context, cmdExecutor executor, ver, name string) (string, error) {
	logger := log.FromContext(ctx)

	var err error
	var bin string

	defer func() {
		if err == nil {
			logger.Debug(fmt.Sprintf("Python binary found: %s", bin))
		}
	}()
	if version.Compare("3.0.0", ver) != version.Greater {
		// looking for python3 or py (windows)
		bin, err = lookForBins(cmdExecutor, "python3", "python3.exe", "py.exe")
		if err != nil {
			return "", fmt.Errorf("%w: %s. Please verify if the executable is included in your PATH", ErrRuntimeNotFound, "python 3")
		}
		vePath, _ := tools.GetPkgVenvPath(name)
		if _, err := os.Stat(vePath); !os.IsNotExist(err) {
			if cmdExecutor.GetOS() == "windows" {
				bin = filepath.Join(vePath, "Scripts", "python.exe")
			} else {
				bin = filepath.Join(vePath, "bin", "python")
			}
		}
		return bin, nil
	}
	if version.Compare("2.0.0", ver) != version.Greater {
		// looking for python2 or py (windows) - no virtualenv
		bin, err = lookForBins(cmdExecutor, "python2", "python2.exe", "py.exe")
		if err != nil {
			return "", fmt.Errorf("%w: %s. Please verify if the executable is included in your PATH", ErrRuntimeNotFound, "python 2")
		}
		return bin, nil
	}
	// looking for any version
	bin, err = lookForBins(cmdExecutor, "python2", "python", "python3", "py.exe", "python.exe")
	if err != nil {
		return "", fmt.Errorf("%w: %s. Please verify if the executable is included in your PATH", ErrRuntimeNotFound, "python")
	}
	return bin, nil
}

func findPipBin(ctx context.Context, cmdExecutor executor, requiredPy string) (string, error) {
	logger := log.FromContext(ctx)

	var bin string
	var err error
	defer func() {
		if err == nil {
			logger.Debug(fmt.Sprintf("Pip binary found: %s", bin))
		}
	}()
	switch version.Compare(requiredPy, "3.0.0") {
	case version.Greater, version.Equals:
		bin, err = lookForBins(cmdExecutor, "pip3", "pip3.exe")
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrPackageManagerNotFound, "pip3")
		}
	case version.Smaller:
		bin, err = lookForBins(cmdExecutor, "pip2")
		if err != nil {
			return "", fmt.Errorf("%w, %s", ErrPackageManagerNotFound, "pip2")
		}
	}

	return bin, nil
}

func installPythonDepsPip(ctx context.Context, cmdExecutor executor, bin, dir string) error {
	logger := log.FromContext(ctx)

	if ok, _ := cmdExecutor.FileExists(filepath.Join(dir, "requirements.txt")); !ok {
		return nil
	}
	logger.Info("requirements.txt found, running pip package manager")

	if err := os.Setenv("PYTHONUSERBASE", dir); err != nil {
		return err
	}
	args := []string{bin, "install", "--user", "--ignore-installed", "-r", filepath.Join(dir, "requirements.txt")}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if _, err := cmdExecutor.ExecCommand(cmd); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			logger.Debug(fmt.Sprintf("Unable execute package manager (PYTHONUSERBASE=%s %s): \n %s", dir, strings.Join(args, " "), exitErr.Stderr))
		}
		return fmt.Errorf("%w: %s. Please verify pip system dependencies (setuptools, python3-dev, gcc, libffi-dev, openssl-dev)", ErrPackageManagerExec, "pip")
	}
	return nil
}

func lookForBins(cmdExecutor executor, bins ...string) (string, error) {
	var err error
	var bin string
	for _, binName := range bins {
		bin, err = cmdExecutor.LookPath(binName)
		if err == nil {
			return bin, nil
		}
	}
	return bin, err
}
