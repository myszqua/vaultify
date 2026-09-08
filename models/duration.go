package models

import (
	"encoding/json"
	"time"
)

// Duration wraps time.Duration with JSON unmarshaling support.
type Duration time.Duration

// UnmarshalJSON parses a duration string (e.g. "5m") from JSON.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(dur)

	return nil
}
