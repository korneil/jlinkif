package internal

import (
	"bytes"
	"github.com/mingrammer/cfmt"
	"github.com/mitchellh/go-ps"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func isGDBRunning() ps.Process {
	procs, err := ps.Processes()
	if err != nil {
		panic(err)
	}

	for _, p := range procs {
		if strings.HasPrefix(p.Executable(), "JLinkGDBServer") {
			return p
		}
	}
	return nil
}

func (x *ctx) runGDBServer() {
	prevPid := 0
	x.wg.Add(1)
	go func() {
		for x.context.Err() == nil {
			p := isGDBRunning()
			if p != nil {
				if prevPid != p.Pid() {
					prevPid = p.Pid()
					cfmt.Warningf("GDB is already running (pid: %d)\n", prevPid)
				}
				time.Sleep(time.Second)
				continue
			}

			prevPid = 0

			cmd := exec.CommandContext(x.context, "JLinkGDBServer", "-if", "SWD", "-device", "NRF52")
			// prevent ctrl+c killing the gdb server
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: true,
			}

			errBuf := bytes.Buffer{}
			cmd.Stderr = &errBuf
			err := cmd.Start()
			cfmt.Warningf("Starting GDB server (pid %d)\n", cmd.Process.Pid)
			if err == nil {
				err = cmd.Wait()
				if x.context.Err() != nil {
					break
				}
			}
			if err != nil {
				errLines := strings.Split(errBuf.String(), "\n")
				for i, line := range errLines {
					errLines[i] = "    " + line
				}
				cfmt.Errorf("GDB server: %v:\n%s\n", err, strings.Join(errLines, "\n"))
			}

			time.Sleep(time.Second)
		}
		x.wg.Done()
	}()
}
