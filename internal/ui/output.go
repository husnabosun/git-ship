// Package ui renders StepResult values to the terminal. It is the only
// package allowed to call fmt.Println for status output — engine and
// preflight packages return data, they never print it themselves.
package ui

import (
	"fmt"

	"github.com/husnabosun/git-ship/internal/core"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// Info prints a neutral, informational line.
func Info(msg string) {
	fmt.Printf("%s[i]%s %s\n", colorCyan, colorReset, msg)
}

// Warn prints a non-fatal warning.
func Warn(msg string) {
	fmt.Printf("%s[!]%s %s\n", colorYellow, colorReset, msg)
}

// Fail prints a fatal error.
func Fail(msg string) {
	fmt.Printf("%s[x]%s %s\n", colorRed, colorReset, msg)
}

// StepResult renders the outcome of a single Step in a consistent format.
func StepOutcome(name string, r core.StepResult) {
	switch {
	case r.Skipped:
		fmt.Printf("%s[-]%s %s %s(skipped: %s)%s\n", colorGray, colorReset, name, colorGray, r.Message, colorReset)
	case r.Success:
		fmt.Printf("%s[✓]%s %s %s\n", colorGreen, colorReset, name, dim(r.Message))
	default:
		fmt.Printf("%s[x]%s %s failed: %v\n", colorRed, colorReset, name, r.Err)
	}
}

func dim(msg string) string {
	if msg == "" {
		return ""
	}
	return colorGray + "— " + msg + colorReset
}
