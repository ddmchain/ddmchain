package soap

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (

	localLoc = time.Local
)

func MarshalUi1(v uint8) (string, error) {
	return strconv.FormatUint(uint64(v), 10), nil
}

func UnmarshalUi1(s string) (uint8, error) {
	v, err := strconv.ParseUint(s, 10, 8)
	return uint8(v), err
}

func MarshalUi2(v uint16) (string, error) {
	return strconv.FormatUint(uint64(v), 10), nil
}

func UnmarshalUi2(s string) (uint16, error) {
	v, err := strconv.ParseUint(s, 10, 16)
	return uint16(v), err
}

func MarshalUi4(v uint32) (string, error) {
	return strconv.FormatUint(uint64(v), 10), nil
}

func UnmarshalUi4(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint32(v), err
}

func MarshalI1(v int8) (string, error) {
	return strconv.FormatInt(int64(v), 10), nil
}

func UnmarshalI1(s string) (int8, error) {
	v, err := strconv.ParseInt(s, 10, 8)
	return int8(v), err
}

func MarshalI2(v int16) (string, error) {
	return strconv.FormatInt(int64(v), 10), nil
}

func UnmarshalI2(s string) (int16, error) {
	v, err := strconv.ParseInt(s, 10, 16)
	return int16(v), err
}

func MarshalI4(v int32) (string, error) {
	return strconv.FormatInt(int64(v), 10), nil
}

func UnmarshalI4(s string) (int32, error) {
	v, err := strconv.ParseInt(s, 10, 32)
	return int32(v), err
}

func MarshalInt(v int64) (string, error) {
	return strconv.FormatInt(v, 10), nil
}

func UnmarshalInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func MarshalR4(v float32) (string, error) {
	return strconv.FormatFloat(float64(v), 'G', -1, 32), nil
}

func UnmarshalR4(s string) (float32, error) {
	v, err := strconv.ParseFloat(s, 32)
	return float32(v), err
}

func MarshalR8(v float64) (string, error) {
	return strconv.FormatFloat(v, 'G', -1, 64), nil
}

func UnmarshalR8(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	return float64(v), err
}

func MarshalFixed14_4(v float64) (string, error) {
	if v >= 1e14 || v <= -1e14 {
		return "", fmt.Errorf("soap fixed14.4: value %v out of bounds", v)
	}
	return strconv.FormatFloat(v, 'f', 4, 64), nil
}

func UnmarshalFixed14_4(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if v >= 1e14 || v <= -1e14 {
		return 0, fmt.Errorf("soap fixed14.4: value %q out of bounds", s)
	}
	return v, nil
}

func MarshalChar(v rune) (string, error) {
	if v == 0 {
		return "", errors.New("soap char: rune 0 is not allowed")
	}
	return string(v), nil
}

func UnmarshalChar(s string) (rune, error) {
	if len(s) == 0 {
		return 0, errors.New("soap char: got empty string")
	}
	r, n := utf8.DecodeRune([]byte(s))
	if n != len(s) {
		return 0, fmt.Errorf("soap char: value %q is not a single rune", s)
	}
	return r, nil
}

func MarshalString(v string) (string, error) {
	return v, nil
}

func UnmarshalString(v string) (string, error) {
	return v, nil
}

func parseInt(s string, err *error) int {
	v, parseErr := strconv.ParseInt(s, 10, 64)
	if parseErr != nil {
		*err = parseErr
	}
	return int(v)
}

var dateRegexps = []*regexp.Regexp{

	regexp.MustCompile(`^(\d{4})(?:-(\d{2})(?:-(\d{2}))?)?$`),

	regexp.MustCompile(`^(\d{4})(?:(\d{2})(?:(\d{2}))?)?$`),
}

func parseDateParts(s string) (year, month, day int, err error) {
	var parts []string
	for _, re := range dateRegexps {
		parts = re.FindStringSubmatch(s)
		if parts != nil {
			break
		}
	}
	if parts == nil {
		err = fmt.Errorf("soap date: value %q is not in a recognized ISO8601 date format", s)
		return
	}

	year = parseInt(parts[1], &err)
	month = 1
	day = 1
	if len(parts[2]) != 0 {
		month = parseInt(parts[2], &err)
		if len(parts[3]) != 0 {
			day = parseInt(parts[3], &err)
		}
	}

	if err != nil {
		err = fmt.Errorf("soap date: %q: %v", s, err)
	}

	return
}

var timeRegexps = []*regexp.Regexp{

	regexp.MustCompile(`^(\d{2})(?::(\d{2})(?::(\d{2}))?)?$`),

	regexp.MustCompile(`^(\d{2})(?:(\d{2})(?:(\d{2}))?)?$`),
}

func parseTimeParts(s string) (hour, minute, second int, err error) {
	var parts []string
	for _, re := range timeRegexps {
		parts = re.FindStringSubmatch(s)
		if parts != nil {
			break
		}
	}
	if parts == nil {
		err = fmt.Errorf("soap time: value %q is not in ISO8601 time format", s)
		return
	}

	hour = parseInt(parts[1], &err)
	if len(parts[2]) != 0 {
		minute = parseInt(parts[2], &err)
		if len(parts[3]) != 0 {
			second = parseInt(parts[3], &err)
		}
	}

	if err != nil {
		err = fmt.Errorf("soap time: %q: %v", s, err)
	}

	return
}

var timezoneRegexp = regexp.MustCompile(`^([+-])(\d{2})(?::?(\d{2}))?$`)

func parseTimezone(s string) (offset int, err error) {
	if s == "Z" {
		return 0, nil
	}
	parts := timezoneRegexp.FindStringSubmatch(s)
	if parts == nil {
		err = fmt.Errorf("soap timezone: value %q is not in ISO8601 timezone format", s)
		return
	}

	offset = parseInt(parts[2], &err) * 3600
	if len(parts[3]) != 0 {
		offset += parseInt(parts[3], &err) * 60
	}
	if parts[1] == "-" {
		offset = -offset
	}

	if err != nil {
		err = fmt.Errorf("soap timezone: %q: %v", s, err)
	}

	return
}

var completeDateTimeZoneRegexp = regexp.MustCompile(`^([^T]+)(?:T([^-+Z]+)(.+)?)?$`)

func splitCompleteDateTimeZone(s string) (dateStr, timeStr, zoneStr string, err error) {
	parts := completeDateTimeZoneRegexp.FindStringSubmatch(s)
	if parts == nil {
		err = fmt.Errorf("soap date/time/zone: value %q is not in ISO8601 datetime format", s)
		return
	}
	dateStr = parts[1]
	timeStr = parts[2]
	zoneStr = parts[3]
	return
}

func MarshalDate(v time.Time) (string, error) {
	return v.In(localLoc).Format("2006-01-02"), nil
}

var dateFmts = []string{"2006-01-02", "20060102"}

func UnmarshalDate(s string) (time.Time, error) {
	year, month, day, err := parseDateParts(s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, localLoc), nil
}

type TimeOfDay struct {

	FromMidnight time.Duration

	HasOffset bool

	Offset int
}

func MarshalTimeOfDay(v TimeOfDay) (string, error) {
	d := int64(v.FromMidnight / time.Second)
	hour := d / 3600
	d = d % 3600
	minute := d / 60
	second := d % 60

	return fmt.Sprintf("%02d:%02d:%02d", hour, minute, second), nil
}

func UnmarshalTimeOfDay(s string) (TimeOfDay, error) {
	t, err := UnmarshalTimeOfDayTz(s)
	if err != nil {
		return TimeOfDay{}, err
	} else if t.HasOffset {
		return TimeOfDay{}, fmt.Errorf("soap time: value %q contains unexpected timezone")
	}
	return t, nil
}

func MarshalTimeOfDayTz(v TimeOfDay) (string, error) {
	d := int64(v.FromMidnight / time.Second)
	hour := d / 3600
	d = d % 3600
	minute := d / 60
	second := d % 60

	tz := ""
	if v.HasOffset {
		if v.Offset == 0 {
			tz = "Z"
		} else {
			offsetMins := v.Offset / 60
			sign := '+'
			if offsetMins < 1 {
				offsetMins = -offsetMins
				sign = '-'
			}
			tz = fmt.Sprintf("%c%02d:%02d", sign, offsetMins/60, offsetMins%60)
		}
	}

	return fmt.Sprintf("%02d:%02d:%02d%s", hour, minute, second, tz), nil
}

func UnmarshalTimeOfDayTz(s string) (tod TimeOfDay, err error) {
	zoneIndex := strings.IndexAny(s, "Z+-")
	var timePart string
	var hasOffset bool
	var offset int
	if zoneIndex == -1 {
		hasOffset = false
		timePart = s
	} else {
		hasOffset = true
		timePart = s[:zoneIndex]
		if offset, err = parseTimezone(s[zoneIndex:]); err != nil {
			return
		}
	}

	hour, minute, second, err := parseTimeParts(timePart)
	if err != nil {
		return
	}

	fromMidnight := time.Duration(hour*3600+minute*60+second) * time.Second

	if fromMidnight > 24*time.Hour || minute >= 60 || second >= 60 {
		return TimeOfDay{}, fmt.Errorf("soap time.tz: value %q has value(s) out of range", s)
	}

	return TimeOfDay{
		FromMidnight: time.Duration(hour*3600+minute*60+second) * time.Second,
		HasOffset:    hasOffset,
		Offset:       offset,
	}, nil
}

func MarshalDateTime(v time.Time) (string, error) {
	return v.In(localLoc).Format("2006-01-02T15:04:05"), nil
}

func UnmarshalDateTime(s string) (result time.Time, err error) {
	dateStr, timeStr, zoneStr, err := splitCompleteDateTimeZone(s)
	if err != nil {
		return
	}

	if len(zoneStr) != 0 {
		err = fmt.Errorf("soap datetime: unexpected timezone in %q", s)
		return
	}

	year, month, day, err := parseDateParts(dateStr)
	if err != nil {
		return
	}

	var hour, minute, second int
	if len(timeStr) != 0 {
		hour, minute, second, err = parseTimeParts(timeStr)
		if err != nil {
			return
		}
	}

	result = time.Date(year, time.Month(month), day, hour, minute, second, 0, localLoc)
	return
}

func MarshalDateTimeTz(v time.Time) (string, error) {
	return v.Format("2006-01-02T15:04:05-07:00"), nil
}

func UnmarshalDateTimeTz(s string) (result time.Time, err error) {
	dateStr, timeStr, zoneStr, err := splitCompleteDateTimeZone(s)
	if err != nil {
		return
	}

	year, month, day, err := parseDateParts(dateStr)
	if err != nil {
		return
	}

	var hour, minute, second int
	var location *time.Location = localLoc
	if len(timeStr) != 0 {
		hour, minute, second, err = parseTimeParts(timeStr)
		if err != nil {
			return
		}
		if len(zoneStr) != 0 {
			var offset int
			offset, err = parseTimezone(zoneStr)
			if offset == 0 {
				location = time.UTC
			} else {
				location = time.FixedZone("", offset)
			}
		}
	}

	result = time.Date(year, time.Month(month), day, hour, minute, second, 0, location)
	return
}

func MarshalBoolean(v bool) (string, error) {
	if v {
		return "1", nil
	}
	return "0", nil
}

func UnmarshalBoolean(s string) (bool, error) {
	switch s {
	case "0", "false", "no":
		return false, nil
	case "1", "true", "yes":
		return true, nil
	}
	return false, fmt.Errorf("soap boolean: %q is not a valid boolean value", s)
}

func MarshalBinBase64(v []byte) (string, error) {
	return base64.StdEncoding.EncodeToString(v), nil
}

func UnmarshalBinBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func MarshalBinHex(v []byte) (string, error) {
	return hex.EncodeToString(v), nil
}

func UnmarshalBinHex(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

func MarshalURI(v *url.URL) (string, error) {
	return v.String(), nil
}

func UnmarshalURI(s string) (*url.URL, error) {
	return url.Parse(s)
}
