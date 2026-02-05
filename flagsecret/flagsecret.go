package flagsecret

import (
	"os"
	"strings"
)

type Secret struct {
	value string
}

func (Secret) String() string {
	return "[REDACTED]"
}
func (s Secret) Value() string {
	return s.value
}

// Reads from a file if the value is prefixed with file://
// otherwise just sets the value to the string provided
func (s *Secret) Set(v string) error {
	if after, ok := strings.CutPrefix(v, "file://"); ok {
		path := after
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s.value = strings.TrimSpace(string(b))
		return nil
	}
	s.value = v
	return nil
}
