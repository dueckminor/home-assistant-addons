package alphaess

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/alphaess"
)

// CSVImportSession stores parsed CSV data in memory
type CSVImportSession struct {
	Date     time.Time
	Timezone string
	Records  []CSVRecord
}

type CSVRecord struct {
	Time                  time.Time
	BatterySOC            float64 // %
	ConsumerLoad          float64 // kW
	BatteryPV             float64 // kW (Energiespeicher PV)
	GridCoupledPV         float64 // kW (PV von netzgekoppeltem PV-Wechselrichter)
	ToGrid                float64 // kW (Einspeisung)
	FromGrid              float64 // kW (Netzbezug)
	SolarProduction       float64 // Calculated: BatteryPV + GridCoupledPV
	BatteryPower          float64 // Calculated
	BatteryCharge         float64 // Calculated (Wh for the interval)
	BatteryDischarge      float64 // Calculated (Wh for the interval)
	BatteryChargeFromGrid float64 // Calculated (Wh for the interval)
}

// Global session storage (in production, use Redis or similar)
var (
	importSessions = make(map[string]*CSVImportSession)
	sessionMutex   sync.RWMutex
)

// ParseCSV parses AlphaESS CSV format and returns parsed records
func ParseCSV(content, dateStr, timezone string) (*CSVImportSession, error) {
	// Parse the date
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = ','
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("CSV file is empty or missing header")
	}

	// Validate header
	header := rows[0]
	expectedHeader := []string{"Datum", "BAT", "Verbraucherlast", "Energiespeicher PV", "PV von netzgekoppeltem PV-Wechselrichter", "Einspeisung", "Netzbezug"}
	if !equalHeaders(header, expectedHeader) {
		return nil, fmt.Errorf("invalid CSV header format")
	}

	session := &CSVImportSession{
		Date:     date,
		Timezone: timezone,
		Records:  make([]CSVRecord, 0, len(rows)-1),
	}

	var prevRecord *CSVRecord

	for i, row := range rows[1:] {
		if len(row) < 7 {
			return nil, fmt.Errorf("row %d has insufficient columns", i+2)
		}

		// Parse time (format: "H:MM" or "HH:MM")
		timeStr := strings.TrimSpace(row[0])
		parts := strings.Split(timeStr, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid time format at row %d: %s", i+2, timeStr)
		}
		hour, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid hour at row %d: %s", i+2, timeStr)
		}
		minute, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid minute at row %d: %s", i+2, timeStr)
		}

		// Handle 24:00 as end of day
		if hour == 24 {
			continue // Skip 24:00 entries
		}

		recordTime := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, loc)

		record := CSVRecord{
			Time:          recordTime,
			BatterySOC:    parseFloat(row[1]),
			ConsumerLoad:  parseFloat(row[2]),
			BatteryPV:     parseFloat(row[3]),
			GridCoupledPV: parseFloat(row[4]),
			ToGrid:        parseFloat(row[5]),
			FromGrid:      parseFloat(row[6]),
		}

		// Calculate solar production
		record.SolarProduction = record.BatteryPV + record.GridCoupledPV

		// Calculate battery power (positive = charging, negative = discharging)
		// Energy balance: Solar + Grid = Consumer Load + To Grid + Battery Charge
		// Battery Power = Solar + Grid - Consumer Load - To Grid
		record.BatteryPower = record.SolarProduction + record.FromGrid - record.ConsumerLoad - record.ToGrid

		// Calculate charge/discharge energy for this 5-minute interval
		intervalHours := 5.0 / 60.0 // 5 minutes = 1/12 hour

		if record.BatteryPower > 0 {
			record.BatteryCharge = record.BatteryPower * intervalHours * 1000 // kW * h = kWh -> Wh
			record.BatteryDischarge = 0

			// Calculate charge from grid
			// If charging and we have grid consumption and solar isn't enough
			if record.FromGrid > 0 {
				solarAvailable := record.SolarProduction - record.ConsumerLoad - record.ToGrid
				if solarAvailable < record.BatteryPower {
					gridToBattery := record.BatteryPower - solarAvailable
					if gridToBattery > 0 {
						record.BatteryChargeFromGrid = gridToBattery * intervalHours * 1000
					}
				}
			}
		} else if record.BatteryPower < 0 {
			record.BatteryDischarge = -record.BatteryPower * intervalHours * 1000
			record.BatteryCharge = 0
			record.BatteryChargeFromGrid = 0
		}

		prevRecord = &record
		session.Records = append(session.Records, record)
	}

	_ = prevRecord // Suppress unused warning

	return session, nil
}

// AggregateToHourly aggregates 5-minute records to hourly values
func (s *CSVImportSession) AggregateToHourly() []alphaess.Measurement {
	if len(s.Records) == 0 {
		return nil
	}

	hourlyData := make(map[time.Time]*struct {
		count                 int
		batterySOC            float64
		toGrid                float64
		fromGrid              float64
		solarProduction       float64
		batteryCharge         float64
		batteryDischarge      float64
		batteryChargeFromGrid float64
	})

	// Aggregate by hour
	for _, record := range s.Records {
		hourTime := time.Date(record.Time.Year(), record.Time.Month(), record.Time.Day(),
			record.Time.Hour(), 0, 0, 0, record.Time.Location())

		if hourlyData[hourTime] == nil {
			hourlyData[hourTime] = &struct {
				count                 int
				batterySOC            float64
				toGrid                float64
				fromGrid              float64
				solarProduction       float64
				batteryCharge         float64
				batteryDischarge      float64
				batteryChargeFromGrid float64
			}{}
		}

		data := hourlyData[hourTime]
		data.count++
		data.batterySOC += record.BatterySOC
		data.toGrid += record.ToGrid
		data.fromGrid += record.FromGrid
		data.solarProduction += record.SolarProduction
		data.batteryCharge += record.BatteryCharge
		data.batteryDischarge += record.BatteryDischarge
		data.batteryChargeFromGrid += record.BatteryChargeFromGrid
	}

	// Convert to measurement values
	// Group all hourly data by metric name
	measurementData := map[string][]alphaess.MeasurementValue{
		"battery_soc":              {},
		"to_grid":                  {},
		"from_grid":                {},
		"solar_production":         {},
		"battery_charge":           {},
		"battery_discharge":        {},
		"battery_charge_from_grid": {},
	}

	for hourTime, data := range hourlyData {
		if data.count > 0 {
			// For rates (kW), take average; for energy (Wh), take sum
			measurementData["battery_soc"] = append(measurementData["battery_soc"], alphaess.MeasurementValue{
				Time:  hourTime.UTC(),
				Value: data.batterySOC / float64(data.count),
			})
			measurementData["to_grid"] = append(measurementData["to_grid"], alphaess.MeasurementValue{
				Time:  hourTime.UTC(),
				Value: data.toGrid * 1000, // kW -> W
			})
			measurementData["from_grid"] = append(measurementData["from_grid"], alphaess.MeasurementValue{
				Time:  hourTime.UTC(),
				Value: data.fromGrid * 1000, // kW -> W
			})
			measurementData["solar_production"] = append(measurementData["solar_production"], alphaess.MeasurementValue{
				Time:  hourTime.UTC(),
				Value: data.solarProduction * 1000, // kW -> W
			})
			measurementData["battery_charge"] = append(measurementData["battery_charge"], alphaess.MeasurementValue{
				Time:  hourTime.UTC(),
				Value: data.batteryCharge, // Already in Wh
			})
			measurementData["battery_discharge"] = append(measurementData["battery_discharge"], alphaess.MeasurementValue{
				Time:  hourTime.UTC(),
				Value: data.batteryDischarge, // Already in Wh
			})
			measurementData["battery_charge_from_grid"] = append(measurementData["battery_charge_from_grid"], alphaess.MeasurementValue{
				Time:  hourTime.UTC(),
				Value: data.batteryChargeFromGrid,
			})
		}
	}

	// Convert to Measurement array
	var results []alphaess.Measurement
	for name, values := range measurementData {
		if len(values) > 0 {
			results = append(results, alphaess.Measurement{
				Name:   name,
				Values: values,
			})
		}
	}

	return results
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func equalHeaders(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != b[i] {
			return false
		}
	}
	return true
}

// StoreSession stores a parsed CSV session in memory
func StoreSession(sessionID string, session *CSVImportSession) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	importSessions[sessionID] = session
}

// GetSession retrieves a stored session
func GetSession(sessionID string) *CSVImportSession {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()
	return importSessions[sessionID]
}

// ClearSession removes a session from memory
func ClearSession(sessionID string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	delete(importSessions, sessionID)
}

// GetAllSessions returns all stored sessions
func GetAllSessions() map[string]*CSVImportSession {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()
	sessions := make(map[string]*CSVImportSession, len(importSessions))
	for k, v := range importSessions {
		sessions[k] = v
	}
	return sessions
}
