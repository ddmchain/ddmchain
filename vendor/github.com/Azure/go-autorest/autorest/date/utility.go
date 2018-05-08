package date

import (
	"strings"
	"time"
)

func ParseTime(format string, t string) (d time.Time, err error) {
	return time.Parse(format, strings.ToUpper(t))
}
