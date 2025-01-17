// Package color provides translation of the colors to the monochromatic representation
package color

import (
	"fmt"

	"github.com/fatih/color"
)

// RedString is a convenient helper function to return a string with reverse video attribute.
func RedString(format string, a ...interface{}) string { return ReverseVideoString(format, a...) }

// GreenString is a convenient helper function to return a string with bold attribute.
func GreenString(format string, a ...interface{}) string { return BoldString(format, a...) }

// YellowString is a convenient helper function to return a string with italic attribute.
func YellowString(format string, a ...interface{}) string { return ItalicString(format, a...) }

// BlueString is a convenient helper function to return a string with faint attribute.
func BlueString(format string, a ...interface{}) string { return FaintString(format, a...) }

// CyanString is a convenient helper function to return a string with underline attribute.
func CyanString(format string, a ...interface{}) string { return UnderlineString(format, a...) }

// HiBlackString is a convenient helper function to return a string with normal attribute.
func HiBlackString(format string, a ...interface{}) string { return fmt.Sprintf(format, a...) }

// HiYellowString is a convenient helper function to return a string with italic attribute.
func HiYellowString(format string, a ...interface{}) string { return ItalicString(format, a...) }

var (
	bold      = color.New(color.Bold)
	italic    = color.New(color.Italic)
	reverse   = color.New(color.ReverseVideo)
	underline = color.New(color.Underline)
	faint     = color.New(color.Faint)
)

// BoldString is a convenient helper function to return a string with bold text
func BoldString(format string, a ...interface{}) string {
	if len(a) == 0 {
		return bold.Sprint(format)
	}
	return bold.Sprintf(format, a...)
}

// ItalicString is a convenient helper function to return a string with italic text
func ItalicString(format string, a ...interface{}) string {
	if len(a) == 0 {
		return italic.Sprint(format)
	}
	return italic.Sprintf(format, a...)
}

// ReverseVideoString is a convenient helper function to return a string with reversed video
func ReverseVideoString(format string, a ...interface{}) string {
	if len(a) == 0 {
		return reverse.Sprint(format)
	}
	return reverse.Sprintf(format, a...)
}

// UnderlineString is a convenient helper function to return a string with underline text
func UnderlineString(format string, a ...interface{}) string {
	if len(a) == 0 {
		return underline.Sprint(format)
	}
	return underline.Sprintf(format, a...)
}

// FaintString is a convenient helper function to return a string with faint (lighter) text
func FaintString(format string, a ...interface{}) string {
	if len(a) == 0 {
		return faint.Sprint(format)
	}
	return faint.Sprintf(format, a...)
}
