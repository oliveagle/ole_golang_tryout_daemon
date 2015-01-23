// +build windows

package main

import (
	"code.google.com/p/winsvc/svc"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	port = ":9977"
)

type myservice struct{}

func (this *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	log.Println("myservice.Execute\r\n")
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	fasttick := time.Tick(500 * time.Millisecond)
	slowtick := time.Tick(2 * time.Second)
	tick := fasttick

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// execute Run
	go serveConn(args, r, changes)

	// major loop for signal processing.
loop:
	for {
		select {
		case <-tick:
			beep()
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				tick = slowtick
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				tick = fasttick
			default:
				log.Printf("unexpected control request #%d", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(name string, isDebug bool) {
	f, err := os.OpenFile("c:\\tools\\myservice.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v \n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	log.Printf("runService: starting %s service \r\n", name)
	err = svc.Run(name, &myservice{})
	if err != nil {
		log.Printf("runService: Error: %s service failed: %v\r\n", name, err)
		return
	}
	log.Printf("runService: %s service stopped\r\n", name)
}

func serveConn(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, error) {
	log.Println("serveConn\r\n")

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Set up listener for defined host and port
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return false, err
	}

	// set up channel on which to send accepted connections
	listen := make(chan net.Conn, 100)
	go acceptConnection(listener, listen)

	// loop work cycle with accept connections or interrupt
	// by system signal
	log.Println("Manage() loop\r\n")
	for {
		select {
		case conn := <-listen:
			go handleClient(conn)
		case killSignal := <-interrupt:
			log.Println("Got signal:", killSignal, "\r\n")
			log.Println("Stoping listening on ", listener.Addr(), "\r\n")
			listener.Close()
			if killSignal == os.Interrupt {
				return false, fmt.Errorf("Daemon was interruped by system signal")
			}
			return false, fmt.Errorf("Daemon was killed")
		}
	}
	return true, nil
}
