package watcher

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

type Config struct {
	Exclude []string `yaml:"exclude"`
	Include []string `yaml:"include"`
}

type Context struct {
	Config Config

	// Listen for ctrl+C
	CloseOnUserInterrupt bool `yaml:"-"`

	// Triggers on file change
	Changed chan string `yaml:"-"`

	context context.Context
	cancel  context.CancelFunc
	watcher *fsnotify.Watcher

	signals chan os.Signal
}

func (x *Context) Init() (err error) {
	x.Changed = make(chan string)
	x.signals = make(chan os.Signal, 2)

	if x.watcher, err = fsnotify.NewWatcher(); err != nil {
		return
	}

	if x.CloseOnUserInterrupt {
		signal.Notify(x.signals, os.Interrupt, syscall.SIGTERM)
	}

	return
}

func (ctx *Context) IsExcluded(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, pattern := range ctx.Config.Exclude {
		for i := 0; i < len(abs); i++ {
			if abs[i] == '/' {
				if m, _ := filepath.Match(pattern, abs[i+1:]); m {
					return true
				}
			}
		}
	}
	return false
}

func (ctx *Context) UpdateWatchedFiles() (err error) {
	for _, f := range ctx.Config.Include {
		err = filepath.Walk(f, func(path string, info os.FileInfo, err error) error {
			if info != nil && info.IsDir() {
				if ctx.IsExcluded(path) {
					return filepath.SkipDir
				}
				ctx.watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			return
		}
	}

	return
}

func (ctx *Context) Start() (err error) {
	if err = ctx.UpdateWatchedFiles(); err != nil {
		return
	}

	go func() {
		defer ctx.Close()

		for {
			select {
			case event, ok := <-ctx.watcher.Events:
				if !ok {
					return
				}
				if event.Op != fsnotify.Chmod {
					// fmt.Printf("changed %s\n", event.Name)
					if !ctx.IsExcluded(event.Name) {
						ctx.Changed <- event.Name
					}
				}
			case <-ctx.signals:
				return
			case _, ok := <-ctx.watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return
}

func (ctx *Context) Close() {
	ctx.watcher.Close()
	close(ctx.Changed)
}

func setFileLimit(n uint64) (err error) {
	var limit syscall.Rlimit
	if err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		return
	}
	limit.Cur = n
	if err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		return
	}
	return syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit)
}
