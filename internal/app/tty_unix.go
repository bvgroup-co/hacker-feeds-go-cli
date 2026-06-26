//go:build linux || darwin

package app

import (
	"os"
	"syscall"
	"unsafe"
)

func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	var termios syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), ioctlReadTermios, uintptr(unsafe.Pointer(&termios)))
	return errno == 0
}
