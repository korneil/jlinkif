package internal

import (
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"syscall"
)

type WatcherConfig struct {
	Root    string   `yaml:"root"`
	Exclude []string `yaml:"exclude"`
	Include []string `yaml:"include"`
}

func (x *ctx) runWatcher() (err error) {
	root := x.config.Watcher.Root
	if root, err = filepath.Abs(root); err != nil {
		return
	}
	fileMap := NewFileMap(root, x.config.Watcher.Include, x.config.Watcher.Exclude)

	x.wg.Add(1)
	go func() {
		defer x.wg.Done()
		for {
			select {
			case <-x.context.Done():
				return
			case event, ok := <-x.watcher.Events:
				if !ok {
					return
				}
				if event.Op != fsnotify.Chmod {
					if fileMap.ToInclude(event.Name) {
						x.triggerBuild <- true
					}
				}
			case _, ok := <-x.watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info != nil && info.IsDir() {
			if fileMap.ExplicitlyExcluded(path) {
				return filepath.SkipDir
			}
			return x.watcher.Add(path)
		}
		return nil
	})
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
