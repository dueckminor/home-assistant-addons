package alphaess

import (
	"fmt"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/alphaess"
)

type MeasurementAggregate struct {
	StartTime             time.Time `json:"start_time"`               // Start of period in UTC
	EndTime               time.Time `json:"end_time"`                 // End of period in UTC
	LocalDate             string    `json:"local_date,omitempty"`     // e.g., "2026-01-18" for daily
	LocalHour             *int      `json:"local_hour,omitempty"`     // 0-23 for hourly (omitted for daily+)
	Gap                   bool      `json:"gap"`                      // True if data is missing/interpolated
	Load                  float64   `json:"load"`                     // Wh
	FromGrid              float64   `json:"from_grid"`                // Wh
	ToGrid                float64   `json:"to_grid"`                  // Wh
	SolarProduction       float64   `json:"solar_production"`         // Wh
	BatteryDischarge      float64   `json:"battery_discharge"`        // Wh
	BatteryCharge         float64   `json:"battery_charge"`           // Wh
	BatteryChargeFromGrid float64   `json:"battery_charge_from_grid"` // Wh
	BatterySOC            float64   `json:"battery_soc"`              // % (average)
}

type AggregateParameters struct {
	Interval string    `json:"interval"` // "hourly", "daily", "monthly"
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	Timezone string    `json:"timezone"` // e.g., "Europe/Berlin", "UTC" (default)
}

func (a *addon) Aggregate(params AggregateParameters) ([]MeasurementAggregate, error) {
	// Load timezone
	loc, err := time.LoadLocation(params.Timezone)
	if err != nil {
		loc = time.UTC
	}

	// Step 1: Get raw measurements
	// as time frame use all measurements between from and to (but not from and to themselves)
	// and include the siblings. If everything is fine, we get just the measurements
	// we need, but in case of gaps, we also get the previous measurement before 'from'
	// and the next measurement after 'to' to be able to calculate deltas.
	filter := MeasurementFilter{
		After:    params.From,
		Before:   params.To,
		Siblings: true,
	}
	measurements := a.getMeasurementsFromDB(filter)

	// Step 2: Convert measurements to aggregates (raw values, EndTime only)
	aggregates := a.measurementsToAggregates(measurements, loc)

	// Step 3: Create time ranges based on interval
	var timeRanges []MeasurementAggregate
	switch params.Interval {
	case "hourly":
		timeRanges = a.aggregateHourly(params.From, params.To, loc)
	case "daily":
		timeRanges = a.aggregateDaily(params.From, params.To, loc)
	case "monthly":
		timeRanges = a.aggregateMonthly(params.From, params.To, loc)
	default:
		return nil, fmt.Errorf("unsupported interval: %s", params.Interval)
	}

	// Step 4: Group aggregates by time ranges and calculate deltas
	result := a.groupAndCalculate(aggregates, timeRanges)

	return result, nil
}

// Convert []Measurement to []MeasurementAggregate with raw values
func (a *addon) measurementsToAggregates(measurements []alphaess.Measurement, loc *time.Location) []MeasurementAggregate {
	// Find all unique timestamps
	timestampMap := make(map[time.Time]*MeasurementAggregate)

	for _, m := range measurements {
		for _, v := range m.Values {
			if _, exists := timestampMap[v.Time]; !exists {
				localTime := v.Time.In(loc)
				hour := localTime.Hour()
				timestampMap[v.Time] = &MeasurementAggregate{
					EndTime:   v.Time,
					LocalDate: localTime.Format("2006-01-02"),
					LocalHour: &hour,
				}
			}
		}
	}

	// Fill in the values
	for _, m := range measurements {
		for _, v := range m.Values {
			agg := timestampMap[v.Time]
			switch m.Name {
			case "from_grid":
				agg.FromGrid = v.Value
			case "to_grid":
				agg.ToGrid = v.Value
			case "solar_production":
				agg.SolarProduction = v.Value
			case "battery_discharge":
				agg.BatteryDischarge = v.Value
			case "battery_charge":
				agg.BatteryCharge = v.Value
			case "battery_charge_from_grid":
				agg.BatteryChargeFromGrid = v.Value
			case "battery_soc":
				agg.BatterySOC = v.Value
			}
		}
	}

	// Convert map to sorted slice
	var result []MeasurementAggregate
	for _, agg := range timestampMap {
		result = append(result, *agg)
	}

	// Sort by time
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].EndTime.After(result[j].EndTime) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

func (a *addon) groupAndCalculate(aggregates []MeasurementAggregate, timeRanges []MeasurementAggregate) []MeasurementAggregate {
	var result []MeasurementAggregate

	for i, agg := range aggregates {
		fmt.Printf("EndTime %d: %s\n", i, agg.EndTime)
	}

	for _, tr := range timeRanges {
		fmt.Printf("\nProcessing TimeRange: %s - %s\n", tr.StartTime, tr.EndTime)
		if tr.StartTime.After(aggregates[len(aggregates)-1].EndTime) {
			break
		}

		agg := MeasurementAggregate{
			StartTime: tr.StartTime,
			EndTime:   tr.EndTime,
			LocalDate: tr.LocalDate,
			LocalHour: tr.LocalHour,
		}

		// Find all aggregates in this time range
		before, after := findAggregateForTime(tr, aggregates)

		if before == nil || after == nil {
			agg.Gap = true
		} else {
			fmt.Printf("Using Measurement:    %s - %s\n", before.EndTime, after.EndTime)

			measermentTimeDiff := after.EndTime.Sub(before.EndTime)
			expectedTimeDiff := tr.EndTime.Sub(tr.StartTime)
			if measermentTimeDiff > expectedTimeDiff*3/2 {
				agg.Gap = true
			}
			factor := expectedTimeDiff.Seconds() / measermentTimeDiff.Seconds()

			agg.FromGrid = (after.FromGrid - before.FromGrid) * factor
			agg.ToGrid = (after.ToGrid - before.ToGrid) * factor
			agg.SolarProduction = (after.SolarProduction - before.SolarProduction) * factor
			agg.BatteryDischarge = (after.BatteryDischarge - before.BatteryDischarge) * factor
			agg.BatteryCharge = (after.BatteryCharge - before.BatteryCharge) * factor
			agg.BatteryChargeFromGrid = (after.BatteryChargeFromGrid - before.BatteryChargeFromGrid) * factor
			agg.BatterySOC = (after.BatterySOC + before.BatterySOC) / 2
		}

		// Calculate load
		agg.Load = agg.SolarProduction + agg.FromGrid + agg.BatteryDischarge - agg.ToGrid - agg.BatteryCharge

		result = append(result, agg)
	}

	return result
}

func findAggregateForTime(timeRange MeasurementAggregate, aggregates []MeasurementAggregate) (before *MeasurementAggregate, after *MeasurementAggregate) {
	for _, agg := range aggregates {
		if !agg.EndTime.After(timeRange.StartTime) {
			before = &agg
		} else if !agg.EndTime.Before(timeRange.EndTime) {
			after = &agg
			break
		}
	}
	return before, after
}
