package packages

import (
	"context"
	"errors"

	"github.com/akamai/cli/pkg/log"
)

type (
	// LangManager allows operations on different programming languages
	LangManager interface {
		Install(ctx context.Context, dir string, requirements LanguageRequirements, commands []string) error
		FindExec(ctx context.Context, requirements LanguageRequirements, cmdExec string) ([]string, error)
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

// Install builds and installs contents of a directory based on provided language requirements
func (l *langManager) Install(ctx context.Context, dir string, reqs LanguageRequirements, commands []string) error {
	lang, requirements := determineLangAndRequirements(reqs)
	switch lang {
	case PHP:
		return l.installPHP(ctx, dir, requirements)
	case Javascript:
		return l.installJavaScript(ctx, dir, requirements)
	case Ruby:
		return l.installRuby(ctx, dir, requirements)
	case Python:
		return l.installPython(ctx, dir, requirements)
	case Go:
		return l.installGolang(ctx, dir, requirements, commands)
	}
	return ErrUnknownLang
}

// FindExec locates language's CLI executable
func (l *langManager) FindExec(ctx context.Context, reqs LanguageRequirements, cmdExec string) ([]string, error) {
	logger := log.FromContext(ctx)
	lang, requirements := determineLangAndRequirements(reqs)
	// FIXME: Add support for other languages defined in readme: Ruby and PHP
	switch lang {
	case Go:
		return []string{cmdExec}, nil
	case Javascript:
		bin, err := l.commandExecutor.LookPath("node")
		if err != nil {
			bin, err = l.commandExecutor.LookPath("nodejs")
		}
		return []string{bin, cmdExec}, nil
	case Python:
		bin, err := findPythonBin(ctx, l.commandExecutor, requirements)
		if err != nil {
			return nil, err
		}
		return []string{bin, cmdExec}, nil
	case Undefined:
		logger.Debugf("command language is not defined")
		return []string{cmdExec}, nil
	default:
		return []string{cmdExec}, nil
	}
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
		return Python, reqs.Python
	}

	return Undefined, ""
}
