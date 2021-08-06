package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func TestCommandsLocator(t *testing.T) {
	require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
	res := CommandLocator(context.Background())
	for i := 0; i < len(res)-1; i++ {
		assert.True(t, strings.Compare(res[i].Name, res[i+1].Name) == -1)
	}
}

func TestSubcommandsToCliCommands_packagePrefix(t *testing.T) {
	from := subcommands{
		Commands: []command{{
			Name:         "testCmd",
			AutoComplete: false,
		}},
		Requirements: packages.LanguageRequirements{Python: "3.0.0"},
		Action:       nil,
		Pkg:          "testPkg",
	}

	cmds := subcommandToCliCommands(from, &git.Mock{}, &packages.Mock{})

	for _, cmd := range cmds {
		assert.True(t, strings.HasPrefix(cmd.Aliases[0], fmt.Sprintf("%s/", from.Pkg)), "there should be an alias with the package prefix")
	}
}
