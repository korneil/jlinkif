package internal

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/mingrammer/cfmt"
	"regexp"
	"strconv"
	"strings"
)

var jlinkHeaderRE = []*regexp.Regexp{
	regexp.MustCompile(`^SEGGER J-Link (.*?) - Real time terminal output$`),
	regexp.MustCompile(`^J-Link .*? compiled .*?, SN=(\d+)$`),
	regexp.MustCompile(`^Process: (JLinkGDBServer\w*)$`),
}

var stripAnsiColorsRE = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

type entry struct {
	Timestamp string
	Level     string
	App       string
	Fnc       string

	binaryLen int
	visual    *logVisualSetting
}

func stripAnsiColors(str string) string {
	return stripAnsiColorsRE.ReplaceAllString(str, "")
}

func NewEntry() (x *entry) {
	x = &entry{}
	return
}

var logLineDateRE = regexp.MustCompile(`^\[(\d{2}:\d{2}:\d{2}.\d{3},\d{3})\]`)
var logLineLevelRE = regexp.MustCompile(`^<(inf|dbg|err|wrn)>`)
var logLineAppRE = regexp.MustCompile(`^[\w]+`)
var logLineFncRE = regexp.MustCompile(`^[\w]+`)
var logLineDumpRE = regexp.MustCompile(`^((?:(?:[0-9a-f]{2}\s)+)\s(?:(?:[0-9a-f]{2}\s|\s)*)|\s+)\|(.*)$`)

var logLeveFormatSettings = map[string]*logVisualSetting{
	"":    {indicator: " ", indicatorColor: 252, msgColor: 252},
	"dbg": {indicator: "⚭", indicatorColor: 39, msgColor: 39},
	"inf": {indicator: "ℹ", indicatorColor: 42, msgColor: 248},
	"wrn": {indicator: "⚠", indicatorColor: 180, msgColor: 180},
	"err": {indicator: "ϟ", indicatorColor: 124, msgColor: 124},
}

func trimPlusOne(s string, l int) string {
	if len(s) > l {
		l++
	}
	return s[l:]
}

func (x *entry) Write(line string) {
	b := strings.Builder{}

	line = stripAnsiColors(line)

	if len(line) > 49 && line[49] == '|' {
		if dump := logLineDumpRE.FindStringSubmatch(line); len(dump) > 0 {
			asciiLen := 0
			hex := dump[1]
			for i := len(hex) - 1; i >= 0; i-- {
				if hex[i] >= '0' && hex[i] <= '9' || hex[i] >= 'a' && hex[i] <= 'f' {
					asciiLen++
				}
			}
			asciiLen /= 2
			prefix := fmt.Sprintf("%s‥%s", toSubscript(x.binaryLen), toSubscript(x.binaryLen+asciiLen-1))
			prefix = fmt.Sprintf("%20s ➜", prefix)
			x.binaryLen += asciiLen
			b.WriteString(aurora.BgGray(2, aurora.Gray(12, prefix)).String() + " ")

			b.WriteString(x.visual.Coloredf(hex))
			b.WriteString(aurora.BrightBlack(" | ").String())
			b.WriteString(x.visual.Coloredf(dump[2]))
		}
	} else {
		x.binaryLen = 0

		if matches := logLineDateRE.FindStringSubmatch(line); len(matches) > 0 {
			ts := matches[0][1 : len(matches[0])-1]
			seconds := 0
			for i := 0; i < 3; i++ {
				seconds *= 60
				if t, err := strconv.Atoi(ts[i*3 : i*3+2]); err == nil {
					seconds += t
				}
			}
			ts = ts[strings.IndexByte(ts, '.')+1:]
			x.Timestamp = fmt.Sprintf("%4d.%s%s", seconds, ts[:3], ts[4:])
			line = trimPlusOne(line, len(matches[0]))
		} else {
			x.Timestamp = ""
		}

		if matches := logLineLevelRE.FindStringSubmatch(line); len(matches) > 0 {
			x.Level = matches[1]
			x.visual = logLeveFormatSettings[x.Level]
			line = trimPlusOne(line, len(matches[0]))
		} else {
			x.Level = ""
		}

		if x.App = logLineAppRE.FindString(line); len(x.App) > 0 {
			isDot := len(line) > len(x.App) && line[len(x.App)] == '.'

			line = trimPlusOne(line, len(x.App))
			if isDot {
				if x.Fnc = logLineFncRE.FindString(line); len(x.Fnc) > 0 {
					line = trimPlusOne(line, len(x.Fnc))
				} else {
					x.Fnc = ""
				}
			}
		} else {
			x.App = ""
		}

		x.writeHeader(&b)

		if b.Len() > 0 {
			b.WriteString(aurora.Gray(12, ":").String())
			if len(line) > 0 && line[0] != ' ' {
				b.WriteByte(' ')
			}
		}

		b.WriteString(x.visual.Coloredf(line))
	}

	fmt.Println(b.String())
}

func (x *entry) writeHeader(b *strings.Builder) {
	if x.Timestamp != "" {
		b.WriteString(aurora.BgGray(2, aurora.Gray(12, x.Timestamp)).String())
	}

	if x.Level != "" {
		b.WriteString(" " + x.visual.Indicator())
	}

	if x.App != "" {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(aurora.BrightBlue(x.App).String())
		if x.Fnc != "" {
			b.WriteString(aurora.Colorize("."+x.Fnc, aurora.CyanFg|aurora.ItalicFm).String())
		}
	}
}

func (x *ctx) RunRTTReader() {
	x.wg.Add(1)
	go func() {
		defer x.wg.Done()

		r := NewRTT(x.context, ":19021")
		defer r.Close()

		jlinkHeaderLines := make([]string, len(jlinkHeaderRE))
		jlinkHeaderState := 0

		sn := ""

		entry := NewEntry()

		for x.context.Err() == nil {
			select {
			case <-x.context.Done():
				break
			case d := <-r.data:
				if d[0] == 13 { // strip \r if it is the first byte
					d = d[1:]
				}
				line := string(d)
				re := jlinkHeaderRE[jlinkHeaderState].FindStringSubmatch(line)
				if len(re) > 0 {
					if len(re) > 1 {
						sn = re[1]
					}
					jlinkHeaderLines[jlinkHeaderState] = line
					jlinkHeaderState++
					if jlinkHeaderState == len(jlinkHeaderRE) {
						cfmt.Successf("RTT Connected (SN: %s)\n", sn)
						jlinkHeaderState = 0
					}
					continue
				} else if jlinkHeaderState != 0 {
					x.mu.Lock()
					p := x.rttPrint
					x.mu.Unlock()
					if p {
						for i := 0; i < jlinkHeaderState-1; i++ {
							entry.Write(jlinkHeaderLines[i])
						}
					}
					jlinkHeaderState = 0
				}

				x.mu.Lock()
				p := x.rttPrint
				x.mu.Unlock()
				if p {
					entry.Write(line)
				}
			}
		}
	}()
}
