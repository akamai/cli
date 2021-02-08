package packages

import (
	"context"
	"errors"
	"github.com/akamai/cli/pkg/log"
	"os/exec"
)

type (
	LangManager interface {
		Install(ctx context.Context, dir string, requirements LanguageRequirements, commands []string) error
		FindExec(ctx context.Context, requirements LanguageRequirements, cmdExec string) ([]string, error)
	}

	Language string

	LanguageRequirements struct {
		Go     string `json:"go"`
		Php    string `json:"php"`
		Node   string `json:"node"`
		Ruby   string `json:"ruby"`
		Python string `json:"python"`
	}
)

const (
	Undefined  Language = ""
	PHP        Language = "php"
	Javascript Language = "javascript"
	Ruby       Language = "ruby"
	Python     Language = "python"
	Go         Language = "go"
)

var (
	ErrUnknownLang = errors.New("command language is not defined")
)

type langManager struct{}

func NewLangManager() LangManager {
	return &langManager{}
}

func (l *langManager) Install(ctx context.Context, dir string, reqs LanguageRequirements, commands []string) error {
	lang, requirements := determineLangAndRequirements(reqs)
	switch lang {
	case PHP:
		return installPHP(ctx, dir, requirements)
	case Javascript:
		return installJavaScript(ctx, dir, requirements)
	case Ruby:
		return installRuby(ctx, dir, requirements)
	case Python:
		return installPython(ctx, dir, requirements)
	case Go:
		return installGolang(ctx, dir, requirements, commands)
	}
	return ErrUnknownLang
}

func (l *langManager) FindExec(ctx context.Context, reqs LanguageRequirements, cmdExec string) ([]string, error) {
	logger := log.FromContext(ctx)
	lang, requirements := determineLangAndRequirements(reqs)
	// FIXME: Add support for other languages defined in readme: Ruby and PHP
	switch lang {
	case Go:
		return []string{cmdExec}, nil
	case Javascript:
		bin, err := exec.LookPath("node")
		if err != nil {
			bin, err = exec.LookPath("nodejs")
		}
		return []string{bin, cmdExec}, nil
	case Python:
		bin, err := findPythonBin(ctx, requirements)
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

func determineLangAndRequirements(reqs LanguageRequirements) (Language, string) {
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
