//go:build !windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// ttySize asks the terminal on stdout for its size.
func ttySize() (int, int, bool) {
	var ws struct{ Row, Col, X, Y uint16 }
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, os.Stdout.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	return int(ws.Col), int(ws.Row), e == 0 && ws.Col > 0
}
