package main

type IServiceManager interface {
	InstallService(name string, desc string) error
	RemoveService(name string) error
	RunService(name string, isDebug bool) error

	StartService(name string) error
	StopService(name string) error
	PauseService(name string) error
	ContinueService(name string) error

	Status(name string) string
}

type ServiceManager struct{}
