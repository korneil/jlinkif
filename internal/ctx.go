package internal

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"github.com/mingrammer/cfmt"
	"gopkg.in/yaml.v3"
	"os"
	"os/signal"
	"sync"
)

type Config struct {
	Watcher *WatcherConfig `yaml:"watcher"`
}

type ctx struct {
	config *Config

	context context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	watcher *fsnotify.Watcher
	mu      sync.Mutex

	// build, flash, debug

	bfdInProgress bool
	gdbRunning    bool
	bfdCtx        context.Context
	bfdCancel     context.CancelFunc
	bfdSuccess    chan bool

	signals      chan os.Signal
	triggerBuild chan bool

	// controls whether RTTLines are forwarded to stdout (only while gdb is running)
	rttPrint bool
}

func NewContext(cfg *Config) (x *ctx, err error) {
	x = &ctx{
		config:       cfg,
		bfdSuccess:   make(chan bool),
		signals:      make(chan os.Signal, 2),
		triggerBuild: make(chan bool, 1),
	}

	if x.watcher, err = fsnotify.NewWatcher(); err != nil {
		return
	}

	signal.Notify(x.signals, os.Interrupt, os.Kill)
	x.context, x.cancel = context.WithCancel(context.Background())

	return
}

func (x *ctx) Run() (err error) {
	defer func() {
		if err != nil {
			x.Close()
		}
	}()

	x.runGDBServer()
	x.RunRTTReader()
	if err = x.runWatcher(); err != nil {
		return
	}
	if err = x.runBuilder(); err != nil {
		return
	}

	return nil
}

func (x *ctx) Close() {
	x.cancel()
	if x.watcher != nil {
		_ = x.watcher.Close()
	}
}

func (x *ctx) setFlags(inProgress, gdbRunning, rttPrint bool) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.bfdInProgress = inProgress
	x.gdbRunning = gdbRunning
	x.rttPrint = rttPrint
}

func (x *ctx) getFlags() (inProgress, gdbRunning, rttPrint bool) {
	x.mu.Lock()
	defer x.mu.Unlock()
	inProgress = x.bfdInProgress
	gdbRunning = x.gdbRunning
	rttPrint = x.rttPrint
	return
}

func (x *ctx) Wait() {
signalLoop:
	for {
		select {
		case <-x.signals:
			bfdInProgress, gdbRunning, _ := x.getFlags()
			if bfdInProgress {
				if !gdbRunning {
					x.bfdCancel()
				}
				continue
			}
			break signalLoop
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

func (x *ctx) GetConfigYAML() (o []byte) {
	o, _ = yaml.Marshal(x.config)
	return
}
