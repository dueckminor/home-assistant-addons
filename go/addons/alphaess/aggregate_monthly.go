package alphaess

import (
	"time"
)

func (a *addon) aggregateMonthly(from, to time.Time, loc *time.Location) []MeasurementAggregate {
	// Convert from/to to local timezone
	fromLocal := from.In(loc)
	toLocal := to.In(loc)

	// Start at the beginning of the month
	current := time.Date(fromLocal.Year(), fromLocal.Month(), 1, 0, 0, 0, 0, loc)

	var results []MeasurementAggregate

	for current.Before(toLocal) {
		// Next month
		next := current.AddDate(0, 1, 0)

		if next.After(toLocal) {
			break
		}

		agg := MeasurementAggregate{
			StartTime: current.UTC(),
			EndTime:   next.UTC(),
			LocalDate: current.Format("2006-01"),
		}
		results = append(results, agg)

		current = next
	}

	return results
}
