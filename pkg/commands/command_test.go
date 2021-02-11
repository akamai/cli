package commands

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
	"os"
	"strings"
	"testing"
)

func TestCommandsLocator(t *testing.T) {
	require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
	res := CommandLocator(context.Background())
	for i := 0; i < len(res)-1; i++ {
		assert.True(t, strings.Compare(res[i].Name, res[i+1].Name) == -1)
	}
}
