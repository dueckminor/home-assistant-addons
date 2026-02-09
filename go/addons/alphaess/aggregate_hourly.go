package alphaess

import (
	"time"
)

func (a *addon) aggregateHourly(from, to time.Time, loc *time.Location) []MeasurementAggregate {
	// Convert from/to to local timezone
	fromLocal := from.In(loc)
	toLocal := to.In(loc)

	// Start at the beginning of the hour
	current := time.Date(fromLocal.Year(), fromLocal.Month(), fromLocal.Day(), fromLocal.Hour(), 0, 0, 0, loc)

	var results []MeasurementAggregate

	for current.Before(toLocal) {
		next := current.Add(time.Hour)

		// Handle DST spring forward - skip missing hour
		if next.Sub(current) > time.Hour {
			current = next
			continue
		}

		if next.After(toLocal) {
			break
		}

		hour := current.Hour()
		agg := MeasurementAggregate{
			StartTime: current.UTC(),
			EndTime:   next.UTC(),
			LocalDate: current.Format("2006-01-02"),
			LocalHour: &hour,
		}
		results = append(results, agg)

		current = next
	}

	return results
}
