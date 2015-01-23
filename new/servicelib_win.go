// +build windows

package main

import (
	"code.google.com/p/winsvc/debug"
	"code.google.com/p/winsvc/eventlog"
	"code.google.com/p/winsvc/mgr"
	"code.google.com/p/winsvc/svc"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const (
	port = ":9977"
)

func (this ServiceManager) IsAnInteractiveSession() (bool, error) {
	return svc.IsAnInteractiveSession()
}

func (this ServiceManager) StartService(name string) error {
	// return startService(name)
	log.Println("ServiceManager.StartService\r\n")
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	log.Println("Connected mgr\r\n")

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()

	log.Println("Opened Service\r\n")

	err = s.Start([]string{"p1", "p2", "p3"})
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}
	log.Println("returned ServiceManager.StartService\r\n")
	return nil
}

func (this ServiceManager) InstallService(name, desc string) error {
	log.Println("ServiceManager.InstallService\r\n")
	exepath, err := exePath()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}
	s, err = m.CreateService(name, exepath, mgr.Config{DisplayName: desc})
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %s", err)
	}
	return nil
}

func (this ServiceManager) RemoveService(name string) error {
	log.Println("ServiceManager.RemoveService\r\n")
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}
	return nil
}

func (this ServiceManager) StopService(name string) error {
	log.Println("ServiceManager.StopService\r\n")
	return controlService(name, svc.Stop, svc.Stopped)
}

func (this ServiceManager) PauseService(name string) error {
	log.Println("ServiceManager.PauseService\r\n")
	return controlService(svcName, svc.Pause, svc.Paused)
}

func (this ServiceManager) ContinueService(name string) error {
	log.Println("ServiceManager.ContinueService\r\n")
	return controlService(svcName, svc.Continue, svc.Running)
}

func controlService(name string, c svc.Cmd, to svc.State) error {
	log.Printf("controlService: %s \r\n", name)
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()
	status, err := s.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", c, err)
	}
	timeout := time.Now().Add(10 * time.Second)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", to)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil
}

func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}

var elog debug.Log

type myservice struct{}

func (this *myservice) Run(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, error) {
	log.Println("Run\r\n")

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

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	log.Println("myservice.Execute\r\n")
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	fasttick := time.Tick(500 * time.Millisecond)
	slowtick := time.Tick(2 * time.Second)
	tick := fasttick

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// execute Run
	go m.Run(args, r, changes)

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
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

// func (this ServiceManager) RunService(name string, isDebug bool) {
// 	log.Println("ServiceManager.RunService\r\n")

// 	f, err := os.OpenFile("c:\\tools\\myservice.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
// 	if err != nil {
// 		defer f.Close()
// 		log.SetOutput(f)

// 		log.Println("runService\r\n")

// 		if isDebug {
// 			elog = debug.New(name)
// 		} else {
// 			elog, err = eventlog.Open(name)
// 			if err != nil {
// 				return
// 			}
// 		}
// 		defer elog.Close()

// 		elog.Info(1, fmt.Sprintf("starting %s service", name))
// 		log.Printf("runService: starting %s service \r\n", name)
// 		run := svc.Run
// 		if isDebug {
// 			run = debug.Run
// 		}
// 		err = run(name, &myservice{})
// 		if err != nil {
// 			elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
// 			log.Printf("runService: Error: %s service failed: %v\r\n", name, err)
// 			return
// 		}
// 		elog.Info(1, fmt.Sprintf("%s service stopped", name))
// 		log.Printf("runService: %s service stopped\r\n", name)
// 		fmt.Printf("error opening file: %v \n", err)
// 	}

// }

func runService(name string, isDebug bool) {
	// f, err := os.OpenFile("c:\\tools\\myservice.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	// if err != nil {
	// 	fmt.Printf("error opening file: %v \n", err)
	// }
	// defer f.Close()

	// log.SetOutput(f)

	f, err := os.OpenFile("c:\\tools\\myservice.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v \n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	log.Println("runService\r\n")

	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", name))
	log.Printf("runService: starting %s service \r\n", name)
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &myservice{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		log.Printf("runService: Error: %s service failed: %v\r\n", name, err)
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
	log.Printf("runService: %s service stopped\r\n", name)
}
