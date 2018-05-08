
package uuid

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

func (uuid *UUID) Scan(src interface{}) error {
	switch src.(type) {
	case string:

		if src.(string) == "" {
			return nil
		}

		parsed := Parse(src.(string))

		if parsed == nil {
			return errors.New("Scan: invalid UUID format")
		}

		*uuid = parsed
	case []byte:
		b := src.([]byte)

		if len(b) == 0 {
			return nil
		}

		if len(b) == 16 {
			*uuid = UUID(b)
		} else {
			u := Parse(string(b))

			if u == nil {
				return errors.New("Scan: invalid UUID format")
			}

			*uuid = u
		}

	default:
		return fmt.Errorf("Scan: unable to scan type %T into UUID", src)
	}

	return nil
}

func (uuid UUID) Value() (driver.Value, error) {
	return uuid.String(), nil
}
