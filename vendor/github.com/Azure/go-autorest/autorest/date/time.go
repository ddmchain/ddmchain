package date

import (
	"regexp"
	"time"
)

const (
	azureUtcFormatJSON = `"2006-01-02T15:04:05.999999999"`
	azureUtcFormat     = "2006-01-02T15:04:05.999999999"
	rfc3339JSON        = `"` + time.RFC3339Nano + `"`
	rfc3339            = time.RFC3339Nano
	tzOffsetRegex      = `(Z|z|\+|-)(\d+:\d+)*"*$`
)

type Time struct {
	time.Time
}

func (t Time) MarshalBinary() ([]byte, error) {
	return t.Time.MarshalText()
}

func (t *Time) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

func (t Time) MarshalJSON() (json []byte, err error) {
	return t.Time.MarshalJSON()
}

func (t *Time) UnmarshalJSON(data []byte) (err error) {
	timeFormat := azureUtcFormatJSON
	match, err := regexp.Match(tzOffsetRegex, data)
	if err != nil {
		return err
	} else if match {
		timeFormat = rfc3339JSON
	}
	t.Time, err = ParseTime(timeFormat, string(data))
	return err
}

func (t Time) MarshalText() (text []byte, err error) {
	return t.Time.MarshalText()
}

func (t *Time) UnmarshalText(data []byte) (err error) {
	timeFormat := azureUtcFormat
	match, err := regexp.Match(tzOffsetRegex, data)
	if err != nil {
		return err
	} else if match {
		timeFormat = rfc3339
	}
	t.Time, err = ParseTime(timeFormat, string(data))
	return err
}

func (t Time) String() string {

	b, err := t.MarshalText()
	if err != nil {
		return ""
	}
	return string(b)
}

func (t Time) ToTime() time.Time {
	return t.Time
}
