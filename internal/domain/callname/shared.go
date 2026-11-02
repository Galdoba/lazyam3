package callname

import (
	"regexp"
	"time"
)

func NewPRT() string {
	time.Sleep(time.Second)
	return "PRT" + time.Now().UTC().Format("060102150405")
}

var prtRe = regexp.MustCompile(`^PRT\d{12}$`)
