package config

import (
	"fmt"
	"time"
)

// Duration is a time.Duration that round-trips through TOML as a readable
// string such as "10s". TOML has no native duration type and a bare integer
// would be ambiguous about its unit.
type Duration time.Duration

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Duration) UnmarshalText(text []byte) error {
	s := string(text)
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("%q is not a duration (use e.g. \"10s\", \"1m30s\")", s)
	}
	*d = Duration(parsed)
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// IsZero lets the TOML encoder omit unset per-instance overrides.
func (d Duration) IsZero() bool { return d == 0 }

// String renders the duration.
func (d Duration) String() string { return time.Duration(d).String() }
