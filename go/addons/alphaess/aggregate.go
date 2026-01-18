package alphaess

import "time"

type MeasurementAggregate struct {
	Timestamp             time.Time `json:"timestamp"`
	Gap                   bool      `json:"gap"`
	Load                  float64   `json:"load"`
	FromGrid              float64   `json:"from_grid"`
	ToGrid                float64   `json:"to_grid"`
	SolarProduction       float64   `json:"solar_production"`
	BatteryDischarge      float64   `json:"battery_discharge"`
	BatteryCharge         float64   `json:"battery_charge"`
	BatteryChargeFromGrid float64   `json:"battery_charge_from_grid"`
	BatterySOC            float64   `json:"battery_soc"`
}

type AggregateParameters struct {
	Interval string    `json:"interval"` // e.g., "hourly", "daily"
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
}

func (a *addon) Aggregate() []MeasurementAggregate {
	return nil
}
