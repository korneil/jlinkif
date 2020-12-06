package internal

import (
	"fmt"
	"github.com/logrusorgru/aurora"
)

type logVisualSetting struct {
	indicator        string
	indicatorColor   uint8
	indicatorColorBg uint8
	msgColor         uint8
	msgColorBg       uint8
}

func (x *logVisualSetting) Indicator() string {
	if x == nil {
		return " "
	}
	r := aurora.Reset(x.indicator)
	if x.msgColor > 0 {
		r = r.Index(x.indicatorColor)
	}
	return r.String()
}

func (x *logVisualSetting) Coloredf(format string, a ...interface{}) string {
	r := aurora.Reset(fmt.Sprintf(format, a...))
	if x != nil && x.msgColor > 0 {
		r = r.Index(x.msgColor)
	}
	return r.String()
}

func (x *logVisualSetting) Coloredfln(format string, a ...interface{}) string {
	return x.Coloredf(format, a...) + "\n"
}
