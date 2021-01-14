package io

import (
	"fmt"
	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/version"
	"github.com/fatih/color"
	"strings"
)

func ShowBanner() {
	fmt.Fprintln(app.App.Writer)
	bg := color.New(color.BgMagenta)
	fmt.Fprintf(app.App.Writer, bg.Sprintf(strings.Repeat(" ", 60)+"\n"))
	fg := bg.Add(color.FgWhite)
	title := "Welcome to Akamai CLI v" + version.Version
	ws := strings.Repeat(" ", 16)
	fmt.Fprintf(app.App.Writer, fg.Sprintf(ws+title+ws+"\n"))
	fmt.Fprintf(app.App.Writer, bg.Sprintf(strings.Repeat(" ", 60)+"\n"))
	fmt.Fprintln(app.App.Writer)
}
