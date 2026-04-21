package kafkarest

import (
	"fmt"
	"time"

	_ "time/tzdata"
)

const eventTimeZoneValidCheckName = "event_time_zone_valid"

type eventTimeZoneValidCheck struct{}

func newEventTimeZoneValidCheck(_ Config) (RecordChecker, error) {
	return &eventTimeZoneValidCheck{}, nil
}

func (c *eventTimeZoneValidCheck) Name() string { return eventTimeZoneValidCheckName }

func (c *eventTimeZoneValidCheck) Check(ctx CheckContext, _ *Record) error {
	if ctx.IsTombstone {
		return nil
	}
	if ctx.Envelope == nil {
		return NewValidationError(ctx.Index, c.Name(), "envelope.meta is missing")
	}
	tz := ctx.Envelope.EventTimeZone
	if tz == "" {
		return NewValidationError(ctx.Index, c.Name(), "envelope.meta.eventTimeZone is required")
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return NewValidationError(
			ctx.Index,
			c.Name(),
			fmt.Sprintf("envelope.meta.eventTimeZone %q is not a valid IANA id", tz),
		)
	}
	return nil
}
