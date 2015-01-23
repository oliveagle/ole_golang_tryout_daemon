// +build windows

package main

import (
	"fmt"
	"log"
	"net"
	// "os"
	// "os/signal"
	"syscall"
	"time"
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

func acceptConnection(listener net.Listener, listen chan<- net.Conn) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		listen <- conn
	}
}

func handleClient(client net.Conn) {
	for {
		buf := make([]byte, 4096)
		numbytes, err := client.Read(buf)
		fmt.Printf("numbytes: %d, err: %s, buf: %v \r\n", numbytes, err, buf[:numbytes])
		if numbytes == 0 || err != nil {
			// EOF, close connection
			return
		}
		if numbytes == 2 && buf[0] == 13 && buf[1] == 10 {
			// [13 10]  "\r\n"
		} else {
			now := time.Now()
			str := fmt.Sprintf("%s: %s\r\n", now.Local().Format("15:04:05.999999999"), buf)
			client.Write([]byte(str))
		}
	}
}
