package internal

import (
	"context"
	"github.com/mingrammer/cfmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func (x *ctx) runBuilder() error {
	x.wg.Add(1)
	go func() {
		defer x.wg.Done()

		for {
			select {
			case <-x.context.Done():
				return
			case <-x.triggerBuild:
				go x.bfd()
			case <-x.bfdSuccess:
				x.triggerBuild <- true
			}
		}
	}()

	x.triggerBuild <- true

	return nil
}

func (x *ctx) bfd() {
	var err error
	if x.bfdCtx != nil {
		x.bfdCancel()
	}
	ctx, cancel := context.WithCancel(x.context)
	x.bfdCancel = func() {
		cancel()
		if x.bfdCtx == ctx {
			x.setFlags(false, false, false)
			x.mu.Lock()
			x.bfdCtx = nil
			x.mu.Unlock()
		}
	}
	defer x.bfdCancel()
	x.mu.Lock()
	x.bfdCtx = ctx
	x.mu.Unlock()

	x.setFlags(true, false, false)

	if err = Build(x.bfdCtx, x.config.Watcher.Root); err != nil {
		return
	}
	if x.bfdCtx.Err() != nil {
		return
	}

	x.setFlags(true, true, true)
	if err = Debug(x.bfdCtx, x.config.Watcher.Root); err != nil {
		return
	}
	x.bfdCancel()

	x.bfdSuccess <- true
}

func Build(ctx context.Context, root string) (err error) {
	cfmt.Successln("Building")
	command := exec.CommandContext(ctx, "west", "build")
	command.Env = append(command.Env, os.Environ()...)
	command.Dir = root
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stdout
	if err = command.Run(); err != nil {
		return
	}
	cfmt.Successln("Build completed")

	return nil
}

func Debug(ctx context.Context, root string) (err error) {
	cfmt.Successln("Starting GDB")

	var f *os.File

	if f, err = ioutil.TempFile("", "gdbinit"); err != nil {
		return
	}
	defer os.Remove(f.Name())
	if _, err = f.Write([]byte(gdbInitCmds)); err != nil {
		return
	}
	if err = f.Close(); err != nil {
		return
	}

	args := []string{
		"build/zephyr/zephyr.elf",
		"--nx", "-q", "--ix", f.Name(),
		"-ex", "target remote :2331",
		"-ex", "mon speed 1000",
		"-ex", "monitor halt",
		"-ex", "monitor reset",
		"-ex", "load",
		"-ex", "monitor reset",
		"-ex", "continue",
	}

	command := exec.CommandContext(ctx, "arm-none-eabi-gdb", args...)
	command.Env = append(command.Env, os.Environ()...)
	command.Dir = root
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stdout
	return command.Run()
}

var gdbInitCmds = `
	set pagination off
	set height unlimited
	set output-radix 16
	set confirm off
	set history save on
	set history size 256
	set history remove-duplicates 256
	set history filename ~/.gdb_history
`
