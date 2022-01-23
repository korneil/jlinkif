package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/mingrammer/cfmt"
)

func (x *Context) Build() (err error) {
	cfmt.Successln("Building")
	command := exec.CommandContext(x.bfdCtx, "west", "build")
	command.Env = append(command.Env, os.Environ()...)
	command.Dir = x.Config.Root
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stdout
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err = command.Run(); err != nil {
		return
	}
	cfmt.Successln("Build completed")

	return nil
}

func (x *Context) Debug() (err error) {
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
		fmt.Printf("err: %+v\n", err)
		return
	}

	load := x.Config.SymbolFile
	var latestMTime int64
	for i, lf := range x.loadFiles {
		if p := strings.Index(lf, " "); p >= 0 {
			lf = lf[:p]
		}
		if s, err := os.Stat(lf); err == nil {
			if nano := s.ModTime().UnixNano(); nano > latestMTime {
				latestMTime = nano
				load = x.loadFiles[i]
			}
		}
	}

	args := []string{
		"--nx", "-q", "--ix", f.Name(),
		"-ex", "target remote :2331",
		"-ex", "mon speed 4000",
		"-ex", "monitor halt",
		"-ex", "monitor sleep 500",
		"-ex", "monitor reset",
		"-ex", "monitor sleep 500",
		"-ex", "load " + load,
		"-ex", "symbol-file " + x.Config.SymbolFile,
		"-ex", "monitor sleep 500",
		"-ex", "monitor reset",
		"-ex", "monitor sleep 500",
		"-ex", "continue",
	}

	defer func() { x.gdbClientCommand = nil }()
	x.gdbClientCommand = exec.CommandContext(x.bfdCtx, "arm-none-eabi-gdb", args...)
	x.gdbClientCommand.Env = append(x.gdbClientCommand.Env, os.Environ()...)
	x.gdbClientCommand.Dir = x.Config.Root
	x.gdbClientCommand.Stdin = os.Stdin
	x.gdbClientCommand.Stdout = os.Stdout
	x.gdbClientCommand.Stderr = os.Stderr
	return x.gdbClientCommand.Run()
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
