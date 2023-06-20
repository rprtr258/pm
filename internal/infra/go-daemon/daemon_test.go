package daemon

import (
	"flag"
	"log"
	"os"
	"syscall"
)

func Example() {
	signal := flag.String("s", "", "send signal to daemon")

	handler := func(sig os.Signal) error {
		log.Println("signal:", sig)
		if sig == syscall.SIGTERM {
			return ErrStop
		}
		return nil
	}

	// Define command: command-line arg, system signal and handler
	AddCommand(StringFlag(signal, "term"), syscall.SIGTERM, handler)
	AddCommand(StringFlag(signal, "reload"), syscall.SIGHUP, handler)
	flag.Parse()

	// Define daemon context
	dmn := &Context{
		PidFileName: "/var/run/daemon.pid",
		PidFilePerm: 0o644,
		LogFileName: "/var/log/daemon.log",
		LogFilePerm: 0o640,
		WorkDir:     "/",
		Umask:       0o27,
		abspath:     "",
		pidFile:     nil,
		logFile:     nil,
		nullFile:    nil,
		Chroot:      "",
		Credential:  nil,
		Env:         nil,
		Args:        nil,
	}

	// Send commands if needed
	if len(ActiveFlags()) > 0 {
		d, err := dmn.Search()
		if err != nil {
			log.Fatalln("Unable to find daemon:", err.Error())
		}

		if err := SendCommands(d); err != nil {
			log.Fatalln("Unable send commands:", err.Error())
		}

		return
	}

	// Process daemon operations - send signal if present flag or daemonize
	child, err := dmn.Reborn()
	if err != nil {
		log.Fatalln(err)
	}
	if child != nil {
		return
	}
	defer dmn.Release() //nolint:errcheck // example

	// Run main operation
	go func() {
		var ch chan struct{}
		<-ch
	}()

	if errServe := ServeSignals(); errServe != nil {
		log.Println("Error:", errServe)
	}
}
