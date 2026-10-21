package callname

import (
	"regexp"
	"time"
)

func NewPRT() string {
	return "PRT" + time.Now().UTC().Format("060102150405")
}

var prtRe = regexp.MustCompile(`^PRT\d{12}$`)
