// Example of a daemon with echo service
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	// "bytes"
	"code.google.com/p/winsvc/debug"
	// "code.google.com/p/winsvc/mgr"
	// "code.google.com/p/winsvc/svc"
	"time"
	// "code.google.com/p/winsvc/winapi"
	"github.com/takama/daemon"

	// "code.google.com/p/winsvc/eventlog"
)

const (

	// name of the service, match with executable file name
	name        = "myservice"
	description = "My Echo Service"

	// port which daemon should be listen
	port = ":9977"
)

// Service has embedded daemon
type Service struct {
	daemon.Daemon
}

var elog debug.Log

func (service *Service) Serve() (string, error) {

	log.Println("Serve()\r\n")

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Set up listener for defined host and port
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return "Possibly was a problem with the port binding", err
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
				return "Daemon was interruped by system signal", nil
			}
			return "Daemon was killed", nil
		}
	}
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage() (string, error) {

	log.Printf("Manage(): args: %v \r\n", os.Args)

	usage := "Usage: myservice install | remove | start | stop | status | serve"

	// if received any kind of command, do it
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			log.Printf("starting %s service\r\n", name)
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		case "log":
			log.Println("testing\r\n")
			return "", nil
		case "serve":
			service.Serve()
		default:
			return usage, nil
		}
	} else {
		log.Println("Error: no args")
	}

	// Do something, call your goroutines, etc
	// service.Serve()

	// never happen, but need to complete code
	return usage, nil
}

// Accept a client connection and collect it in a channel
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

func main() {
	f, err := os.OpenFile("c:\\tools\\myservice.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v \n", err)
	}
	defer f.Close()

	log.SetOutput(f)

	log.Println("main\r\n")

	log.Println("daemon.New\r\n")
	srv, err := daemon.New(name, description)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	service := &Service{srv}

	log.Println("service.Manage()\r\n")
	status, err := service.Manage()
	if err != nil {
		fmt.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	fmt.Println(status)

}
