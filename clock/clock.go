package clock

import "time"

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (c RealClock) Now() time.Time {
	return time.Now()
}

// For testing
type StaticClock struct {
	Time time.Time
}

func (c StaticClock) Now() time.Time {
	return c.Time
}
