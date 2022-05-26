package apphelp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestSetup(t *testing.T) {
	testApp := cli.NewApp()
	testApp.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name: "test-flag",
		},
	}
	flagsNumber := len(testApp.Flags)

	Setup(testApp)

	assert.Len(t, testApp.Flags, flagsNumber+1)
	assert.Equal(t, testApp.Flags[flagsNumber], cli.HelpFlag)
	assert.Len(t, testApp.Commands, 1)
}
