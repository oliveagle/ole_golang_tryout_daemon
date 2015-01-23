// +build windows

package main

import (
	"log"
	"syscall"
)

// BUG(brainman): MessageBeep Windows api is broken on Windows 7,
// so this example does not beep when runs as service on Windows 7.

var (
	beepFunc = syscall.MustLoadDLL("user32.dll").MustFindProc("MessageBeep")
)

func beep() {
	log.Println("beep\r\n")
	beepFunc.Call(0xffffffff)
}
