package packages

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"
)

type (
	// LangManager allows operations on different programming languages
	LangManager interface {
		// Install builds and installs contents of a directory based on provided language requirements
		Install(ctx context.Context, dir string, requirements LanguageRequirements, commands, ldFlags []string) error
		// FindExec locates language's CLI executable
		FindExec(ctx context.Context, requirements LanguageRequirements, cmdExec string) ([]string, error)
		// PrepareExecution performs any operation required before the external command invocation
		PrepareExecution(ctx context.Context, languageRequirements LanguageRequirements, dirName string) error
		// FinishExecution perform any required tear down steps for external command invocation
		FinishExecution(ctx context.Context, languageRequirements LanguageRequirements, dirName string)
		// GetShell returns the available shell
		GetShell(goos string) (string, error)
		// GetOS is a wrapper around executor.GetOS()
		GetOS() string
		// FileExists checks if the given path exists
		FileExists(path string) (bool, error)
	}

	// LanguageRequirements contains version requirements for all supported programming languages
	LanguageRequirements struct {
		Go     string `json:"go"`
		Php    string `json:"php"`
		Node   string `json:"node"`
		Ruby   string `json:"ruby"`
		Python string `json:"python"`
	}
)

// language constants
const (
	Undefined  = ""
	PHP        = "php"
	Javascript = "javascript"
	Ruby       = "ruby"
	Python     = "python"
	Go         = "go"
)

// Defined errors
var (
	ErrUnknownLang                   = errors.New("command language is not defined")
	ErrRuntimeNotFound               = errors.New("unable to locate runtime")
	ErrRuntimeNoVersionFound         = errors.New("unable to determine installed version, minimum version required")
	ErrRuntimeMinimumVersionRequired = errors.New("higher version is required to install this command")
	ErrPackageManagerNotFound        = errors.New("unable to locate package manager in PATH")
	ErrPackageManagerExec            = errors.New("unable to execute package manager")
	ErrPackageNeedsReinstall         = errors.New("you must reinstall this package to continue")
	ErrPackageCompileFailure         = errors.New("unable to build binary")
	ErrPackageExecutableNotFound     = errors.New("package executable not found")
	ErrVirtualEnvCreation            = errors.New("unable to create virtual environment")
	ErrVirtualEnvActivation          = errors.New("unable to activate virtual environment")
	ErrVenvNotFound                  = errors.New("venv python package not found. Please verify your setup")
	ErrPipNotFound                   = errors.New("pip not found. Please verify your setup")
	ErrRequirementsTxtNotFound       = errors.New("requirements.txt not found in the subcommand")
	ErrRequirementsInstall           = errors.New("failed to install required python packages from requirements.txt")
	ErrPipUpgrade                    = errors.New("unable to install/upgrade pip")
	ErrPipSetuptoolsUpgrade          = errors.New("unable to execute 'python3 -m pip install --user --no-cache --upgrade pip setuptools'")
	ErrNoExeFound                    = errors.New("no executables found")
	ErrOSNotSupported                = errors.New("OS not supported")
	ErrPythonVersionNotSupported     = errors.New("python version not supported")
	ErrDirectoryCreation             = errors.New("unable to create directory")
)

type langManager struct {
	commandExecutor executor
}

// NewLangManager returns a default language manager
func NewLangManager() LangManager {
	return &langManager{
		&defaultExecutor{},
	}
}

func (l *langManager) GetShell(goos string) (string, error) {
	var pathErr *os.PathError

	switch goos {
	case "windows":
		return "", nil
	case "linux", "darwin":
		sh, err := lookForBins(l.commandExecutor, "bash", "sh")
		if err != nil && errors.As(err, &pathErr) && pathErr.Err != syscall.ENOENT {
			return "", err
		}
		if err == nil {
			return sh, nil
		}
	}

	return "", ErrOSNotSupported
}

func (l *langManager) Install(ctx context.Context, pkgSrcPath string, reqs LanguageRequirements, commands, ldFlags []string) error {
	lang, requirements := determineLangAndRequirements(reqs)
	switch lang {
	case PHP:
		return l.installPHP(ctx, pkgSrcPath, requirements)
	case Javascript:
		return l.installJavaScript(ctx, pkgSrcPath, requirements)
	case Ruby:
		return l.installRuby(ctx, pkgSrcPath, requirements)
	case Python:
		pkgVenvPath, err := tools.GetPkgVenvPath(filepath.Base(pkgSrcPath))
		if err != nil {
			return err
		}
		err = l.installPython(ctx, pkgVenvPath, pkgSrcPath, requirements)
		if err != nil {
			_ = os.RemoveAll(pkgVenvPath)
		}
		return err
	case Go:
		return l.installGolang(ctx, pkgSrcPath, requirements, commands, ldFlags)
	}
	return ErrUnknownLang
}

func (l *langManager) FindExec(ctx context.Context, reqs LanguageRequirements, cmdExec string) ([]string, error) {
	logger := log.FromContext(ctx)
	lang, requirements := determineLangAndRequirements(reqs)
	switch lang {
	case Go:
		return []string{cmdExec}, nil
	case Javascript:
		bin, err := l.commandExecutor.LookPath("node")
		if err != nil {
			bin, err = l.commandExecutor.LookPath("nodejs")
			if err != nil {
				return nil, err
			}
		}
		return []string{bin, cmdExec}, nil
	case Python:
		cmdName := filepath.Base(cmdExec)
		pythonBin, err := findPythonBin(ctx, l.commandExecutor, requirements, cmdName)
		if err != nil {
			return nil, err
		}
		return []string{pythonBin, cmdExec}, nil
	case Undefined:
		logger.Debugf("command language is not defined")
		return []string{cmdExec}, nil
	default:
		return []string{cmdExec}, nil
	}
}

func (l *langManager) PrepareExecution(ctx context.Context, languageRequirements LanguageRequirements, dirName string) error {
	if languageRequirements.Python != "" {
		logger := log.FromContext(ctx)

		logger.Debugf("Validating python dependencies")
		python34Bin, pipBin, err := l.validatePythonDeps(ctx, logger, languageRequirements.Python, dirName)
		if err != nil {
			return err
		}

		srcPath, err := tools.GetAkamaiCliSrcPath()
		if err != nil {
			return err
		}

		pkgSrcPath := filepath.Join(srcPath, dirName)

		pkgVePath, err := tools.GetPkgVenvPath(dirName)
		if err != nil {
			return err
		}

		logger.Debugf("Adding any missing element to the virtual environment")
		return l.setup(ctx, pkgVePath, pkgSrcPath, python34Bin, pipBin, languageRequirements.Python, true)
	}
	return nil
}

func (l *langManager) FinishExecution(ctx context.Context, reqs LanguageRequirements, dirName string) {
	if reqs.Python != "" {
		if version.Compare(reqs.Python, "3.0.0") != version.Smaller {
			venvPath, _ := tools.GetPkgVenvPath(dirName)
			l.deactivateVirtualEnvironment(ctx, venvPath, reqs.Python)
		}
	}
}

func (l *langManager) GetOS() string {
	return l.commandExecutor.GetOS()
}

func determineLangAndRequirements(reqs LanguageRequirements) (string, string) {
	if reqs.Php != "" {
		return PHP, reqs.Php
	}

	if reqs.Node != "" {
		return Javascript, reqs.Node
	}

	if reqs.Ruby != "" {
		return Ruby, reqs.Ruby
	}

	if reqs.Go != "" {
		return Go, reqs.Go
	}

	if reqs.Python != "" {
		return Python, translateWildcards(reqs.Python)
	}

	return Undefined, ""
}

func translateWildcards(version string) string {
	version = strings.ReplaceAll(version, "*", "0")
	splitVersion := strings.Split(version, ".")
	for len(splitVersion) < 3 {
		splitVersion = append(splitVersion, "0")
	}
	return strings.Join(splitVersion, ".")
}

func (l *langManager) FileExists(path string) (bool, error) {
	return l.commandExecutor.FileExists(path)
}
