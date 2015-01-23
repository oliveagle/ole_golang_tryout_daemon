// +build windows

package servicelib

import (
	"code.google.com/p/winsvc/eventlog"
	"code.google.com/p/winsvc/mgr"
	"code.google.com/p/winsvc/svc"
	"fmt"
	"log"
	"time"
)

func IsAnInteractiveSession() (bool, error) {
	return svc.IsAnInteractiveSession()
}

func StartService(name string) error {
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

func InstallService(name, desc string) error {
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

func RemoveService(name string) error {
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

func Status(name string) error {
	log.Println("ServiceManagement.Status\r\n")
	return nil
}

func StopService(name string) error {
	log.Println("ServiceManager.StopService\r\n")
	return controlService(name, svc.Stop, svc.Stopped)
}

func PauseService(name string) error {
	log.Println("ServiceManager.PauseService\r\n")
	return controlService(name, svc.Pause, svc.Paused)
}

func ContinueService(name string) error {
	log.Println("ServiceManager.ContinueService\r\n")
	return controlService(name, svc.Continue, svc.Running)
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
