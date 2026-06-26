package app

import (
	"os"

	"golang.org/x/term"
)

func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
