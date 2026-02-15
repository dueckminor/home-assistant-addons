package alphaess

import (
	"time"
)

func (a *addon) aggregateDaily(from, to time.Time, loc *time.Location) []MeasurementAggregate {
	// Convert from/to to local timezone
	fromLocal := from.In(loc)
	toLocal := to.In(loc)

	// Start at midnight of the first day
	current := time.Date(fromLocal.Year(), fromLocal.Month(), fromLocal.Day(), 0, 0, 0, 0, loc)

	var results []MeasurementAggregate

	for current.Before(toLocal) {
		// Next day at midnight
		next := current.AddDate(0, 0, 1)

		if next.After(toLocal) {
			break
		}

		agg := MeasurementAggregate{
			StartTime: current.UTC(),
			EndTime:   next.UTC(),
			LocalDate: current.Format("2006-01-02"),
		}
		results = append(results, agg)

		current = next
	}

	return results
}
