package packages

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPythonVersionRegexp(t *testing.T) {
	tests := map[string]struct {
		input   string
		isMatch bool
	}{
		"good input - python 3.4": {
			input:   "Python 3.4.0",
			isMatch: true,
		},
		"good input - python 2.7": {
			input:   "Python 2.7.10",
			isMatch: true,
		},
		"good input - trailing characters": {
			input:   "Python 3.4.0    ",
			isMatch: true,
		},
		"good input - trailing line break": {
			input:   "Python 3.4.0\n",
			isMatch: true,
		},
		"bad input - no version": {
			input:   "Python",
			isMatch: false,
		},
		"bad input - no python": {
			input:   "3.4.0",
			isMatch: false,
		},
		"bad input - no space": {
			input:   "Python3.4.0",
			isMatch: false,
		},
		"bad input - rubbish": {
			input:   "random string",
			isMatch: false,
		},
		"bad input - empty": {
			input:   "",
			isMatch: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.isMatch {
				assert.Regexp(t, pythonVersionRegex, test.input)
			} else {
				assert.NotRegexp(t, pythonVersionRegex, test.input)
			}
		})
	}
}

func TestPipVersionRegexp(t *testing.T) {
	tests := map[string]struct {
		input   string
		isMatch bool
	}{
		"good input - python 3.9": {
			input:   "pip 21.1.1 from /usr/local/lib/python3.9/site-packages/pip (python 3.9)",
			isMatch: true,
		},
		"good input - trailing line break": {
			input:   "pip 21.1.1 from /usr/local/lib/python3.9/site-packages/pip (python 3.9)\n",
			isMatch: true,
		},
		"bad input - no version": {
			input:   "pip from /usr/local/lib/python3.9/site-packages/pip (python 3.9)",
			isMatch: false,
		},
		"bad input - no python": {
			input:   "pip 21.1.1 from /usr/local/lib/python3.9/site-packages/pip",
			isMatch: false,
		},
		"bad input - no space": {
			input:   "pip21.1.1 from /usr/local/lib/python3.9/site-packages/pip (python 3.9)",
			isMatch: false,
		},
		"bad input - rubbish": {
			input:   "random string",
			isMatch: false,
		},
		"bad input - empty": {
			input:   "",
			isMatch: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.isMatch {
				assert.Regexp(t, pipVersionRegex, test.input)
			} else {
				assert.NotRegexp(t, pipVersionRegex, test.input)
			}
		})
	}
}

func TestInstallPython(t *testing.T) {
	pip2Bin := "/test/pip2"
	bashBin := "/test/bash"
	py2Bin := "/test/python2"
	py3Bin := "/test/python3"
	py3VeBin := filepath.Join("veDir", "bin", "python")
	py3PipVersion := "pip 21.0.1 from /usr/lib/python3.8/site-packages/pip (python 3.8)"
	py3VenvHelp := `
usage: venv [-h] [--system-site-packages] [--symlinks | --copies] [--clear]
            [--upgrade] [--without-pip] [--prompt PROMPT]
            ENV_DIR [ENV_DIR ...]
venv: error: the following arguments are required: ENV_DIR
`
	activationScript := filepath.Join("veDir", "bin", "activate")
	activationScriptWin := filepath.Join("veDir", "Scripts", "activate.bat")
	py3BinWindows := filepath.Join("c:", "Program files", "Python", "python3.exe")
	py3WindowsPipVersion := "pip 20.1.3 from c:\\Program Files\\WindowsApps\\" +
		"PythonSoftwareFoundation.Python.3.9_3.9.1264.0_x64__qbz5n2kfra8p0\\" +
		"lib\\site-packages\\pip (python 3.4)"
	py310WindowsPipVersion := "pip 22.0.4 from c:\\Python\\lib\\site-packages\\pip (python 3.10)"
	py2Version := "Python 2.7.16"
	py34Version := "Python 3.4.0"
	py310Version := "Python 3.10.0"
	ver2 := "2.0.0"
	ver3 := "3.0.0"
	ver355 := "3.5.5"
	srcDir := "testDir"
	veDir := "veDir"
	requirementsFile := filepath.Join("testDir", "requirements.txt")
	winVePipPath := filepath.Join("veDir", "Scripts", "pip.exe")
	winDeactivatePath := filepath.Join("veDir", "Scripts", "deactivate.bat")

	tests := map[string]struct {
		givenDir   string
		veDir      string
		requiredPy string
		goos       string
		init       func(*mocked)
		withError  error
	}{
		"without python 3, python 3 required": {
			givenDir:   "testDir",
			requiredPy: ver3,
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("", errors.New("")).Once()
				m.On("LookPath", "py.exe").Return("", errors.New("")).Once()
				m.On("LookPath", "python3.exe").Return("", errors.New("")).Once()
			},
			withError: ErrRuntimeNotFound,
		},
		"without python 2, python 2 required": {
			givenDir:   srcDir,
			requiredPy: ver2,
			init: func(m *mocked) {
				m.On("LookPath", "python2").Return("", errors.New("")).Once()
				m.On("LookPath", "py.exe").Return("", errors.New("")).Once()
				m.On("LookPath", "python2.exe").Return("", errors.New("")).Once()
			},
			withError: ErrRuntimeNotFound,
		},
		"with python 3.4 and pip, python 3 required": {
			givenDir:   srcDir,
			veDir:      veDir,
			requiredPy: ver3,
			goos:       "linux",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return(py3Bin, nil).Once()
				m.On("LookPath", "bash").Return(bashBin, nil).Times(3)
				m.On("ExecCommand", &exec.Cmd{
					Path: py3Bin,
					Args: []string{py3Bin, "--version"},
				}, true).Return([]byte(py34Version), nil).Twice()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3Bin,
					Args: []string{py3Bin, "-m", "pip", "--version"},
				}, true).Return([]byte(py3PipVersion), nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3Bin,
					Args: []string{py3Bin, "-m", "venv", "--version"},
				}, true).Return([]byte(py3VenvHelp), nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3Bin,
					Args: []string{py3Bin, "-m", "ensurepip", "--upgrade"},
				}, true).Return(nil, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3Bin,
					Args: []string{py3Bin, "-m", "pip", "install", "--no-cache", "--upgrade", "pip", "setuptools"},
				}, true).Return(nil, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3Bin,
					Args: []string{py3Bin, "-m", "venv", "veDir"},
					Dir:  "",
				}, true).Return(nil, nil).Once()
				m.On("GetOS").Return("linux").Times(4)
				m.On("FileExists", veDir).Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: bashBin,
					Args: []string{"source", activationScript},
					Dir:  "",
				}, true).Return(nil, nil).Once()
				m.On("FileExists", requirementsFile).Return(true, nil).Once()
				m.On("FileExists", ".").Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3VeBin,
					Args: []string{py3VeBin, "-m", "pip", "install", "--upgrade", "--ignore-installed", "-r", requirementsFile},
					Dir:  "",
				}, true).Return(nil, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: bashBin,
					Args: []string{"deactivate"},
				}, true).Return(nil, nil).Once()
			},
		},
		"with py 3.4 (windows) and pip, python 3 required": {
			givenDir:   srcDir,
			veDir:      veDir,
			requiredPy: ver3,
			goos:       "windows",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("", errors.New("")).Once()
				m.On("LookPath", "python3.exe").Return(py3BinWindows, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "-m", "pip", "--version"},
				}, true).Return([]byte(py3WindowsPipVersion), nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "--version"},
				}, true).Return([]byte(py34Version), nil).Twice()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "-m", "venv", "--version"},
				}, true).Return([]byte(py3VenvHelp), nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "-m", "ensurepip", "--upgrade"},
				}, true).Return(nil, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "-m", "pip", "install", "--no-cache", "--upgrade", "pip", "setuptools"},
				}, true).Return(nil, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "-m", "venv", "veDir"},
					Dir:  "",
				}, true).Return(nil, nil).Once()
				m.On("FileExists", veDir).Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: activationScriptWin,
					Args: []string{},
					Dir:  "",
				}, true).Return(nil, nil).Once()
				m.On("GetOS").Return("windows").Times(4)
				m.On("FileExists", requirementsFile).Return(true, nil).Once()
				m.On("FileExists", ".").Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: winVePipPath,
					Args: []string{winVePipPath, "install", "--upgrade", "--ignore-installed", "-r", requirementsFile},
					Dir:  "",
				}, true).Return(nil, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: winDeactivatePath,
					Args: []string{winDeactivatePath},
				}, true).Return(nil, nil).Once()
			},
		},
		"with py 3.10 (windows) and pip, python 3 required": {
			givenDir:   srcDir,
			veDir:      veDir,
			requiredPy: ver3,
			goos:       "windows",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("", errors.New("")).Once()
				m.On("LookPath", "python3.exe").Return(py3BinWindows, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "-m", "pip", "--version"},
				}, true).Return([]byte(py310WindowsPipVersion), nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "--version"},
				}, true).Return([]byte(py310Version), nil).Twice()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "-m", "venv", "--version"},
				}, true).Return([]byte(py3VenvHelp), nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "-m", "ensurepip", "--upgrade"},
				}, true).Return(nil, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "-m", "pip", "install", "--no-cache", "--upgrade", "pip", "setuptools"},
				}, true).Return(nil, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3BinWindows,
					Args: []string{py3BinWindows, "-m", "venv", "veDir"},
					Dir:  "",
				}, true).Return(nil, nil).Once()
				m.On("FileExists", veDir).Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: activationScriptWin,
					Args: []string{},
					Dir:  "",
				}, true).Return(nil, nil).Once()
				m.On("GetOS").Return("windows").Times(4)
				m.On("FileExists", requirementsFile).Return(true, nil).Once()
				m.On("FileExists", ".").Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: winVePipPath,
					Args: []string{winVePipPath, "install", "--upgrade", "--ignore-installed", "-r", requirementsFile},
					Dir:  "",
				}, true).Return(nil, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: winDeactivatePath,
					Args: []string{winDeactivatePath},
				}, true).Return(nil, nil).Once()
			},
		},
		"with python 2, 3.9 and pip, python 2 required": {
			givenDir:   srcDir,
			requiredPy: ver2,
			init: func(m *mocked) {
				m.On("LookPath", "python2").Return(py2Bin, nil).Once()
				m.On("LookPath", "pip2").Return(pip2Bin, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py2Bin,
					Args: []string{py2Bin, "--version"},
				}, true).Return([]byte(py2Version), nil).Once()
				m.On("FileExists", requirementsFile).Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: pip2Bin,
					Args: []string{pip2Bin, "install", "--user", "--ignore-installed", "-r", requirementsFile},
					Dir:  "testDir",
				}).Return([]byte(py2Version), nil).Once()
			},
		},
		"with python 2, python 2 required": {
			givenDir:   srcDir,
			requiredPy: ver2,
			goos:       "linux",
			init: func(m *mocked) {
				m.On("LookPath", "python2").Return(py2Bin, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py2Bin,
					Args: []string{py2Bin, "--version"},
				}, true).Return([]byte(py2Version), nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: pip2Bin,
					Args: []string{pip2Bin, "install", "--user", "--ignore-installed", "-r", requirementsFile},
					Dir:  srcDir,
				}).Return([]byte(py2Version), nil).Once()
				m.On("LookPath", "pip2").Return(pip2Bin, nil).Once()
				m.On("FileExists", requirementsFile).Return(true, nil).Once()
			},
		},
		"with empty required version, error version not supported": {
			givenDir: srcDir,
			veDir:    veDir,
			init: func(m *mocked) {
			},
			withError: ErrPythonVersionNotSupported,
		},
		"version not found": {
			givenDir:   srcDir,
			veDir:      veDir,
			requiredPy: ver3,
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return(py3Bin, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3Bin,
					Args: []string{py3Bin, "--version"},
				}, true).Return([]byte{}, nil).Once()
			},
			withError: ErrRuntimeNoVersionFound,
		},
		"version too low": {
			givenDir:   srcDir,
			veDir:      veDir,
			requiredPy: ver355,
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return(py3Bin, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py3Bin,
					Args: []string{py3Bin, "--version"},
				}, true).Return([]byte(py34Version), nil).Once()
			},
			withError: ErrRuntimeMinimumVersionRequired,
		},
		"python 2 required, pip2 bin not found": {
			givenDir:   srcDir,
			requiredPy: ver2,
			init: func(m *mocked) {
				m.On("LookPath", "python2").Return(py2Bin, nil).Once()
				m.On("LookPath", "pip2").Return("", fmt.Errorf("not found")).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: py2Bin,
					Args: []string{py2Bin, "--version"},
				}, true).Return([]byte(py2Version), nil).Once()
			},
			withError: ErrPackageManagerNotFound,
		},
		"python 2 required, just python 3 is installed": {
			givenDir:   srcDir,
			requiredPy: ver2,
			init: func(m *mocked) {
				m.On("LookPath", "python2").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "python2.exe").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "py.exe").Return(py2Bin, nil).Once()
				m.On("ExecCommand", &exec.Cmd{Path: py2Bin, Args: []string{"/test/python2", "--version"}}, true).Return([]byte(py34Version), nil).Once()
			},
			withError: ErrRuntimeNotFound,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := new(mocked)
			test.init(m)
			l := langManager{m}
			err := l.installPython(context.Background(), test.veDir, test.givenDir, test.requiredPy)
			m.AssertExpectations(t)
			if test.withError != nil {
				assert.True(t, errors.Is(err, test.withError), "want: %s; got: %s", test.withError, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
