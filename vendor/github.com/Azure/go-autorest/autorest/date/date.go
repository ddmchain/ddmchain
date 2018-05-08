
package date

import (
	"fmt"
	"time"
)

const (
	fullDate     = "2006-01-02"
	fullDateJSON = `"2006-01-02"`
	dateFormat   = "%04d-%02d-%02d"
	jsonFormat   = `"%04d-%02d-%02d"`
)

type Date struct {
	time.Time
}

func ParseDate(date string) (d Date, err error) {
	return parseDate(date, fullDate)
}

func parseDate(date string, format string) (Date, error) {
	d, err := time.Parse(format, date)
	return Date{Time: d}, err
}

func (d Date) MarshalBinary() ([]byte, error) {
	return d.MarshalText()
}

func (d *Date) UnmarshalBinary(data []byte) error {
	return d.UnmarshalText(data)
}

func (d Date) MarshalJSON() (json []byte, err error) {
	return []byte(fmt.Sprintf(jsonFormat, d.Year(), d.Month(), d.Day())), nil
}

func (d *Date) UnmarshalJSON(data []byte) (err error) {
	d.Time, err = time.Parse(fullDateJSON, string(data))
	return err
}

func (d Date) MarshalText() (text []byte, err error) {
	return []byte(fmt.Sprintf(dateFormat, d.Year(), d.Month(), d.Day())), nil
}

func (d *Date) UnmarshalText(data []byte) (err error) {
	d.Time, err = time.Parse(fullDate, string(data))
	return err
}

func (d Date) String() string {
	return fmt.Sprintf(dateFormat, d.Year(), d.Month(), d.Day())
}

func (d Date) ToTime() time.Time {
	return d.Time
}
