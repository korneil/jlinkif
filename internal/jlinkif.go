package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/korneil/jlinkif/internal/watcher"
	"github.com/mingrammer/cfmt"
	"github.com/tevino/abool"
	"gopkg.in/yaml.v3"
)

type Context struct {
	Config Config

	context context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex

	// build, flash, debug
	bfdCtx     context.Context
	bfdCancel  context.CancelFunc
	bfdSuccess chan bool

	gdbClientCommand   *exec.Cmd
	signals            chan os.Signal
	triggerBuild       chan bool
	buildDone          chan bool
	gdbServerStarted   chan bool
	buildResultPending bool

	// files to check in order and pick the last modified
	loadFiles []string

	// controls whether RTTLines are forwarded to stdout (only while gdb is running)
	rttPrint abool.AtomicBool

	watcher watcher.Context
}

func Init(x *Context) (err error) {
	x.context, x.cancel = context.WithCancel(context.Background())

	x.bfdCancel = func() {}
	x.bfdSuccess = make(chan bool, 2)
	x.buildDone = make(chan bool, 2)
	x.signals = make(chan os.Signal, 2)
	x.triggerBuild = make(chan bool, 2)

	x.Config.Watch.Include = append([]string{x.Config.Root}, x.Config.Watch.Include...)
	x.watcher.CloseOnUserInterrupt = false
	x.watcher.Config = x.Config.Watch
	if err = x.watcher.Init(); err != nil {
		return
	}

	signal.Notify(x.signals, os.Interrupt, syscall.SIGTERM)

	x.loadFiles = strings.Split(x.Config.Load, "|")
	for i := range x.loadFiles {
		x.loadFiles[i] = strings.TrimSpace(x.loadFiles[i])
	}
	if len(x.loadFiles) == 1 && x.loadFiles[0] == "" {
		x.loadFiles = []string{}
	}

	return
}

func (x *Context) Run() (err error) {
	defer func() {
		if err != nil {
			x.Close()
		}
	}()

	x.RunGDBServer()
	x.RunRTTReader()
	if err = x.watcher.Start(); err != nil {
		return
	}

	x.triggerBuild <- true

	return
}

func (x *Context) Close() {
	x.cancel()
	x.watcher.Close()
}

func (x *Context) Wait() {
	bfdCancelKillCounter := 0
	// gdbClientPid := 0

signalLoop:
	for {
		select {
		case <-x.triggerBuild:
			var err error

			x.bfdCancel()

			ctx, cancel := context.WithCancel(x.context)
			x.bfdCancel = func() {
				cancel()
				if x.bfdCtx == ctx {
					x.mu.Lock()
					x.bfdCtx = nil
					x.bfdCancel = func() {}
					x.mu.Unlock()
				}
			}
			x.mu.Lock()
			x.bfdCtx = ctx
			x.mu.Unlock()

			// x.rttPrint.SetTo(false)
			go func() {
				if err = x.Build(); err != nil {
					fmt.Printf("x.Build(): %+v\n", err)
					return
				}
				x.buildDone <- true
			}()

		case <-x.buildDone:
			x.rttPrint.SetTo(true)
			go func() {
				if err := x.Debug(); err != nil {
					fmt.Printf("x.Debug(): %+v\n", err)
					return
				}
				x.bfdSuccess <- true
			}()

		case <-x.bfdSuccess:
			x.triggerBuild <- true

		case <-x.watcher.Changed:
			x.triggerBuild <- true

		case <-x.signals:
			fmt.Printf("cancelled\n")
			if x.gdbClientCommand != nil {
				// if x.gdbClientCommand.Process.Pid != gdbClientPid {
				// bfdCancelKillCounter = 1
				// gdbClientPid = x.gdbClientCommand.Process.Pid
				// } else {
				// bfdCancelKillCounter++
				// }
				// if bfdCancelKillCounter > 3 {
				// break signalLoop
				// }
				// x.gdbClientCommand.Process.Signal(os.Interrupt)
			} else {
				break signalLoop
			}
			// if !bfdInProgress {
			// 	break
			// }
			// if !gdbRunning {
			// 	x.bfdCancel()
			// }
			// // x.triggerBuild <- true
			// rerun <- true

			bfdCancelKillCounter++

		case <-x.context.Done():
			break signalLoop
		}
	}

	cfmt.Infoln("Shutting down")

	go func() {
		x.cancel()
		x.wg.Wait()
		close(x.signals)
	}()

	killCounter := 3
	if _, ok := <-x.signals; ok {
		killCounter--
		if killCounter == 0 {
			cfmt.Warningln("Killing")
		}
	}
}

func (x *Context) GetConfigYAML() (o []byte) {
	o, _ = yaml.Marshal(&x.Config)
	return
}
