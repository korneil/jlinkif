package internal

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/mingrammer/cfmt"
	"github.com/mitchellh/go-ps"
)

func (x *Context) isGDBRunning() ps.Process {
	procs, err := ps.Processes()
	if err != nil {
		panic(err)
	}

	for _, p := range procs {
		if strings.HasPrefix(p.Executable(), x.Config.GDB.Exec) {
			return p
		}
	}
	return nil
}

func (x *Context) RunGDBServer() {
	var cmd *exec.Cmd
	prevPid := 0
	x.wg.Add(1)
	go func() {
		<-x.context.Done()
		if cmd != nil {
			cfmt.Infof("Gracefully shutting down GDB server\n")
			cmd.Process.Signal(os.Interrupt)
		}
	}()

	go func() {
		errBuf := bytes.Buffer{}

		for x.context.Err() == nil {
			p := x.isGDBRunning()
			if p != nil {
				if prevPid != p.Pid() {
					prevPid = p.Pid()
					cfmt.Warningf("GDB is already running (pid: %d)\n", prevPid)
				}
				time.Sleep(time.Second)
				continue
			}
			prevPid = 0

			errBuf.Reset()

			cmd = exec.Command(x.Config.GDB.Exec, x.Config.GDB.Args...)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // prevent ctrl+c killing the gdb server
			cmd.Stderr = &errBuf
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stdout
			if err := cmd.Start(); err != nil {
				fmt.Printf("err: %+v\n", err)
				time.Sleep(time.Second)
				cmd = nil
				continue
			}
			cfmt.Warningf("Starting GDB server (pid %d)\n", cmd.Process.Pid)
			// x.gdbServerStarted <- true
			if err := cmd.Wait(); err != nil {
				errLines := strings.Split(errBuf.String(), "\n")
				for i, line := range errLines {
					errLines[i] = "    " + line
				}
				cfmt.Errorf("GDB server: %v:\n%s\n", err, strings.Join(errLines, "\n"))
				time.Sleep(time.Second)
			}

			cmd = nil
		}

		if cmd != nil {
			fmt.Printf("Killing GDB\n")
			cmd.Process.Kill()
		}

		x.wg.Done()
	}()
}
