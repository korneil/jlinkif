package internal

// import (
// 	"fmt"
// 	"mynewt.apache.org/newt/util"
// 	"mynewt.apache.org/newtmgr/newtmgr/cli"
// 	"mynewt.apache.org/newtmgr/newtmgr/config"
// 	"mynewt.apache.org/newtmgr/newtmgr/nmutil"
// 	"mynewt.apache.org/newtmgr/nmxact/nmserial"
// 	"os"
// 	"os/signal"
// 	"syscall"
// )

// func stopXport() {
// 	x, err := cli.GetXportIfOpen()
// 	if err == nil {
// 		// Don't attempt to close a serial transport.  Attempting to close
// 		// the serial port while a read is in progress (in MacOS) just
// 		// blocks until the read completes.  Instead, let the OS close the
// 		// port on termination.
// 		if _, ok := x.(*nmserial.SerialXport); !ok {
// 			x.Stop()
// 		}
// 	}
// }

// func closeSesn() {
// 	s, err := cli.GetSesnIfOpen()
// 	if err == nil {
// 		s.Close()
// 	}
// }

// func cleanup() {
// 	closeSesn()
// 	stopXport()
// }

// func t() {
// 	nmutil.ToolInfo = nmutil.ToolInfoType{
// 		ExeName:       "mcumgr",
// 		ShortName:     "mcumgr",
// 		LongName:      "mcumgr",
// 		VersionString: "0.0.0-dev",
// 		CfgFilename:   ".mcumgr.cp.json",
// 	}

// 	if err := config.InitGlobalConnProfileMgr(); err != nil {
// 		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
// 		os.Exit(1)
// 	}

// 	defer cleanup()
// 	cli.SetOnExit(cleanup)

// 	sigChan := make(chan os.Signal, 1)
// 	signal.Notify(sigChan)

// 	go func() {
// 		for {
// 			s := <-sigChan
// 			switch s {
// 			case os.Interrupt, syscall.SIGTERM:
// 				cli.SilenceErrors()
// 				cli.NmExit(1)

// 			case syscall.SIGQUIT:
// 				util.PrintStacks()
// 			}
// 		}
// 	}()

// 	cli.Commands().Execute()
// }
