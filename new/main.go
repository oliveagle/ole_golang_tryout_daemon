package main

import (
	"code.google.com/p/winsvc/svc"
	"fmt"
	"log"
	"os"
	"strings"
)

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       install, remove, debug, start, stop, pause or continue.\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

const (
	svcName = "oleservice win"
)

func main() {

	isIntSess, err := svc.IsAnInteractiveSession()
	// isIntSess, err := svrMgr.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}
	if !isIntSess {
		runService(svcName, false)
		return
	}

	if len(os.Args) < 2 {
		usage("no command specified")
	}

	// log ---------
	f, err := os.OpenFile("c:\\tools\\myservice.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v \n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	svrMgr := ServiceManager{}
	log.Println("new func main\r\n")

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "debug":
		runService(svcName, true)
		return
	case "install":
		err = svrMgr.InstallService(svcName, "my service")
	case "remove":
		err = svrMgr.RemoveService(svcName)
	case "start":
		err = svrMgr.StartService(svcName)
	case "stop":
		err = svrMgr.StopService(svcName)
	case "pause":
		err = svrMgr.PauseService(svcName)
	case "continue":
		err = svrMgr.ContinueService(svcName)
	default:
		usage(fmt.Sprintf("invalid command %s", cmd))
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
	}
	return
}
