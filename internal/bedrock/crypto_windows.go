//go:build windows

package bedrock

import "syscall"

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

func FreeConsole() {
	kernel32.NewProc("FreeConsole").Call()
}
