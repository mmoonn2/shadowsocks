package utils

import (
	"os"
	"os/signal"
	"syscall"
)

// SignalHandleFunc before close will do CloseFunc
type SignalHandleFunc func()

// HandleSignal handler close signal
func HandleSignal(sighupHandler, closeHandler SignalHandleFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for sig := range c {
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			if closeHandler != nil {
				closeHandler()
			}
			return
		case syscall.SIGHUP:
			if sighupHandler != nil {
				sighupHandler()
			}
		}
	}
}
