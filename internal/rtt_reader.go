package internal

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/mingrammer/cfmt"
	"regexp"
	"time"
)

var jlinkHeaderRE = []*regexp.Regexp{
	regexp.MustCompile(`^SEGGER J-Link (.*?) - Real time terminal output$`),
	regexp.MustCompile(`^J-Link .*? compiled .*?, SN=(\d+)$`),
	regexp.MustCompile(`^Process: (JLinkGDBServer\w*)$`),
}

type RTTLine struct {
	Time time.Time
	Line string
	Ctx  interface{}
}

func (x *RTTLine) Print() {
	fmt.Printf("%s: ", aurora.BrightBlack(x.Time.Format("03-04-05.000000")))
	fmt.Println(x.Line)
}

func (x *ctx) RunRTTReader() {
	x.wg.Add(1)
	go func() {
		defer x.wg.Done()

		r := NewRTT(x.context, ":19021")
		defer r.Close()

		jlinkHeaderLines := make([]*RTTLine, len(jlinkHeaderRE))
		jlinkHeaderState := 0

		for x.context.Err() == nil {
			select {
			case <-x.context.Done():
				break
			case d := <-r.data:
				if d[0] == 13 {
					d = d[1:]
				}
				rttLine := &RTTLine{Time: time.Now(), Line: string(d)}
				re := jlinkHeaderRE[jlinkHeaderState].FindStringSubmatch(rttLine.Line)
				if len(re) > 0 {
					if len(re) > 1 {
						rttLine.Ctx = re[1]
					}
					jlinkHeaderLines[jlinkHeaderState] = rttLine
					jlinkHeaderState++
					if jlinkHeaderState == len(jlinkHeaderRE) {
						cfmt.Successf("RTT Connected (SN: %s)\n", jlinkHeaderLines[1].Ctx)
						jlinkHeaderState = 0
					}
					continue
				} else if jlinkHeaderState != 0 {
					for i := 0; i < jlinkHeaderState-1; i++ {
						rttLine.Print()
					}
					jlinkHeaderState = 0
				}

				x.mu.Lock()
				p := x.rttPrint
				x.mu.Unlock()
				if p {
					rttLine.Print()
				}
			}
		}
	}()
}
