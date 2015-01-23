package main

import (
	"fmt"
	"github.com/oliveagle/ole_tryout_daemon/servicelib"
	"log"
	"os"
	"strings"
)

const (
	svcName = "oleservice win"
)

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       install, remove, status, start, stop, pause or continue.\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

func main() {
	isIntSess, err := servicelib.IsAnInteractiveSession()
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

	// - log --------------------
	f, err := os.OpenFile("c:\\tools\\myservice.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v \n", err)
	}
	defer f.Close()
	log.SetOutput(f)
	// -------------------- log -

	log.Println("new func main\r\n")

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "install":
		err = servicelib.InstallService(svcName, "my service")
	case "remove":
		err = servicelib.RemoveService(svcName)
	case "start":
		err = servicelib.StartService(svcName)
	case "stop":
		err = servicelib.StopService(svcName)
	case "pause":
		err = servicelib.PauseService(svcName)
	case "continue":
		err = servicelib.ContinueService(svcName)
	case "status":
		err = servicelib.Status(svcName)
	default:
		usage(fmt.Sprintf("invalid command %s", cmd))
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
	}
	return
}
