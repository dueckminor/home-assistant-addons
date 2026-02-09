package alphaess

import (
	"fmt"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/automation"
	"github.com/simonvetter/modbus"
)

type Config struct {
	URI string `yaml:"uri"`
}

type MeasurementValue struct {
	Value float64   `json:"value"`
	Time  time.Time `json:"time"`
}

type Measurement struct {
	Name   string             `json:"name"`
	Unit   string             `json:"unit,omitempty"`
	Values []MeasurementValue `json:"values"`
}

type MeasurementFilter struct {
	MeasurementNames    []string
	measurementNamesMap map[string]bool
	Previous            bool
	Siblings            bool
	After               time.Time // t >  v
	NotAfter            time.Time // t <= v
	Before              time.Time // t <  v
	NotBefore           time.Time // t >= v
}

func (filter *MeasurementFilter) MatchName(name string) bool {
	if len(filter.MeasurementNames) == 0 {
		return true
	}
	if filter.measurementNamesMap == nil {
		filter.measurementNamesMap = make(map[string]bool)
		for _, n := range filter.MeasurementNames {
			filter.measurementNamesMap[n] = true
		}
	}
	_, exists := filter.measurementNamesMap[name]
	return exists
}

func (filter MeasurementFilter) TimeSpecified() bool {
	return !filter.After.IsZero() ||
		!filter.NotAfter.IsZero() ||
		!filter.Before.IsZero() ||
		!filter.NotBefore.IsZero()
}

func (filter MeasurementFilter) MatchTime(t time.Time) bool {
	if !filter.After.IsZero() && !(t.After(filter.After)) {
		return false
	}
	if !filter.NotBefore.IsZero() && !(t.Equal(filter.NotBefore) || t.After(filter.NotBefore)) {
		return false
	}
	if !filter.Before.IsZero() {
		fmt.Println(t, "<", filter.Before)
		if !(t.Before(filter.Before)) {
			return false
		}
	}
	if !filter.NotAfter.IsZero() && !(t.Equal(filter.NotAfter) || t.Before(filter.NotAfter)) {
		return false
	}
	return true
}

type Scanner interface {
	GetMeasurementInfos(filter MeasurementFilter) []Measurement
	GetMeasurements(filter MeasurementFilter) []Measurement
}

type scanner struct {
	client *modbus.ModbusClient

	registry automation.Registry
	node     automation.Node

	sensors   []*sensor
	sensorMap map[string]*sensor

	sensorSolarProduction     *sensor
	correctionSolarProduction float64
	sensorToGrid              *sensor
}

type sensor struct {
	automation.Sensor
	Name   string
	Unit   string
	Addr   uint16
	Signed bool
	Words  int
	Scale  float64

	Previous MeasurementValue
	Current  MeasurementValue
}

func (s *scanner) init() {
	s.registry = automation.GetRegistry()
	s.node = s.registry.CreateNode("alphaess")

	// -------------------------------------------------------------------- grid
	s.sensor10Wh(0x0010, "to_grid")
	s.sensor10Wh(0x0012, "from_grid")
	// s.sensor1V(0x0014, "grid_voltage_l1")
	// s.sensor1V(0x0015, "grid_voltage_l2")
	// s.sensor1V(0x0016, "grid_voltage_l3")
	// s.sensor100mA(0x0017, "grid_current_l1")
	// s.sensor100mA(0x0018, "grid_current_l2")
	// s.sensor100mA(0x0019, "grid_current_l3")
	// s.sensorHz(0x001a, "grid_freq")
	// s.sensor1W(0x001b, "grid_active_power_l1")
	// s.sensor1W(0x001d, "grid_active_power_l2")
	// s.sensor1W(0x001f, "grid_active_power_l3")
	s.sensor1W(0x0021, "grid_active_power")
	// s.sensor1W(0x0023, "grid_reactive_power_l1")
	// s.sensor1W(0x0025, "grid_reactive_power_l2")
	// s.sensor1W(0x0027, "grid_reactive_power_l3")
	// s.sensor1W(0x0029, "grid_reactive_power")
	// s.sensor1W(0x002b, "grid_apparent_power_l1")
	// s.sensor1W(0x002d, "grid_apparent_power_l2")
	// s.sensor1W(0x002f, "grid_apparent_power_l3")
	// s.sensor1W(0x0031, "grid_apparent_power")
	// ---------------------------------------------------------------- pv meter
	// s.sensor10Wh(0x0090, "total_to_grid")
	// s.sensor10Wh(0x0092, "total_from_grid")

	s.sensor1W(0x00A1, "total_active_power")

	// ----------------------------------------------------------------- battery
	// s.sensor100mV(0x0100, "battery_voltage")
	// s.sensor100mA(0x0101, "battery_current")
	s.sensor100Wh(0x0120, "battery_charge")
	s.sensor100Wh(0x0122, "battery_discharge")
	s.sensor100Wh(0x0124, "battery_charge_from_grid")
	s.sensor1W2byte(0x0126, "battery_power")

	s.sensor1W(0x040C, "inverter_power_total")
	s.sensor1W(0x041A, "inverter_backup_power_total")

	// s.sensor10Wh(0x0720, "inverter_total_pv_energy")

	s.sensor10Wh(0x08D2, "solar_production")
	s.sensorPercent(0x0102, "battery_soc")

	s.correctionSolarProduction = 0.0
	s.sensorToGrid = s.sensorMap["to_grid"]
	s.sensorSolarProduction = s.sensorMap["solar_production"]
}

func (s *scanner) addSensor(sensor *sensor) *sensor {
	s.sensors = append(s.sensors, sensor)
	s.sensorMap[sensor.Name] = sensor
	return sensor
}

func (s *scanner) sensor10Wh(addr uint16, name string) *sensor {
	sensor := &sensor{
		Sensor: s.node.CreateSensor(automation.MakeSensorTemplate(name).
			SetIcon(automation.Icon_Wh).
			SetUnit(automation.Unit_Wh).SetPrecision(0).
			SetStateClass(automation.StateClass_Total).
			SetDeviceClass(automation.DeviceClass_Energy)),
		Name:  name,
		Unit:  automation.Unit_Wh.String(),
		Addr:  addr,
		Words: 2,
		Scale: 10,
	}

	return s.addSensor(sensor)
}

func (s *scanner) sensor100Wh(addr uint16, name string) *sensor {
	sensor := &sensor{
		Sensor: s.node.CreateSensor(automation.MakeSensorTemplate(name).
			SetIcon(automation.Icon_Wh).
			SetUnit(automation.Unit_Wh).SetPrecision(0).
			SetStateClass(automation.StateClass_Total).
			SetDeviceClass(automation.DeviceClass_Energy)),
		Name:  name,
		Unit:  automation.Unit_Wh.String(),
		Addr:  addr,
		Words: 2,
		Scale: 100,
	}
	return s.addSensor(sensor)
}

func (s *scanner) sensor1W(addr uint16, name string) *sensor {
	sensor := &sensor{
		Sensor: s.node.CreateSensor(automation.MakeSensorTemplate(name).
			SetIcon(automation.Icon_W).
			SetUnit(automation.Unit_W).SetPrecision(0).
			SetStateClass(automation.StateClass_Measurement).
			SetDeviceClass(automation.DeviceClass_Power)),
		Name:   name,
		Unit:   automation.Unit_W.String(),
		Addr:   addr,
		Signed: true,
		Words:  2,
		Scale:  1,
	}
	return s.addSensor(sensor)
}

func (s *scanner) sensor1W2byte(addr uint16, name string) *sensor {
	sensor := &sensor{
		Sensor: s.node.CreateSensor(automation.MakeSensorTemplate(name).
			SetIcon(automation.Icon_W).
			SetUnit(automation.Unit_W).SetPrecision(0).
			SetStateClass(automation.StateClass_Measurement).
			SetDeviceClass(automation.DeviceClass_Power)),
		Name:   name,
		Unit:   automation.Unit_W.String(),
		Addr:   addr,
		Signed: true,
		Words:  1,
		Scale:  1,
	}
	return s.addSensor(sensor)
}

func (s *scanner) sensor1V(addr uint16, name string) *sensor {
	sensor := &sensor{
		Sensor: s.node.CreateSensor(automation.MakeSensorTemplate(name).
			SetIcon(automation.Icon_V).
			SetUnit(automation.Unit_V).SetPrecision(0).
			SetStateClass(automation.StateClass_Measurement).
			SetDeviceClass(automation.DeviceClass_Voltage)),
		Name:  name,
		Unit:  automation.Unit_V.String(),
		Addr:  addr,
		Words: 1,
		Scale: 1,
	}
	return s.addSensor(sensor)
}

func (s *scanner) sensor100mV(addr uint16, name string) *sensor {
	sensor := &sensor{
		Sensor: s.node.CreateSensor(automation.MakeSensorTemplate(name).
			SetIcon(automation.Icon_V).
			SetUnit(automation.Unit_V).SetPrecision(1).
			SetStateClass(automation.StateClass_Measurement).
			SetDeviceClass(automation.DeviceClass_Voltage)),
		Name:  name,
		Unit:  automation.Unit_V.String(),
		Addr:  addr,
		Words: 1,
		Scale: 0.1,
	}
	return s.addSensor(sensor)
}

func (s *scanner) sensor100mA(addr uint16, name string) *sensor {
	sensor := &sensor{
		Sensor: s.node.CreateSensor(automation.MakeSensorTemplate(name).
			SetIcon(automation.Icon_A).
			SetUnit(automation.Unit_A).SetPrecision(1).
			SetStateClass(automation.StateClass_Measurement).
			SetDeviceClass(automation.DeviceClass_Current)),
		Name:   name,
		Unit:   automation.Unit_A.String(),
		Addr:   addr,
		Signed: true,
		Words:  1,
		Scale:  0.1,
	}
	return s.addSensor(sensor)
}

func (s *scanner) sensorPercent(addr uint16, name string) *sensor {
	sensor := &sensor{
		Sensor: s.node.CreateSensor(automation.MakeSensorTemplate(name).
			SetIcon(automation.Icon_Battery).
			SetUnit(automation.Unit_Percent).SetPrecision(1).
			SetStateClass(automation.StateClass_Measurement).
			SetDeviceClass(automation.DeviceClass_Battery)),
		Name:  name,
		Unit:  automation.Unit_Percent.String(),
		Addr:  addr,
		Words: 1,
		Scale: 0.1,
	}
	return s.addSensor(sensor)
}

func (s *scanner) modbusConnect() (err error) {
	err = s.client.Open()
	if err != nil {
		return err
	}
	err = s.client.SetUnitId(0x55)
	if err != nil {
		return err
	}
	return nil
}

func (s *scanner) modbusClose() (err error) {
	err = s.client.Close()
	if err != nil {
		fmt.Println("failed to close modbus client:", err)
	}
	return err
}

func Run(uri string) (sc Scanner, err error) {
	s := &scanner{}
	s.sensorMap = make(map[string]*sensor)

	s.client, err = modbus.NewClient(&modbus.ClientConfiguration{
		URL:     uri,
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	s.init()

	go func() {
		err = s.node.Connect()
		if err != nil {
			fmt.Println("node connect failed: ", err)
			return
		}
		defer func() {
			err = s.node.Disconnect()
			if err != nil {
				fmt.Println("failed to disconnect:", err)
			}
		}()

		for {
			s.handleModbus()
			time.Sleep(time.Minute * 1)
		}
	}()

	return s, nil
}

func (s *scanner) handleModbus() {

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("crash in handleModbus")
		}
	}()

	err := s.fetchValues()
	if err != nil {
		fmt.Println("fetchValues failed: ", err)
		return
	}

	s.fixSolarProduction()

	for _, sensor := range s.sensors {
		sensor.SetState(sensor.Current.Value)
	}

	s.sensorSolarProduction.Current.Value -= s.correctionSolarProduction
}

func (s *scanner) fetchValues() error {
	err := s.modbusConnect()
	if err != nil {
		fmt.Println("modbusConnect failed: ", err)
		return err
	}

	defer func() {
		err = s.modbusClose()
		if err != nil {
			fmt.Println("failed to close modbus client:", err)
		}
	}()

	now := time.Now().UTC()

	for _, sensor := range s.sensors {
		var value int64
		if sensor.Words == 1 {
			var value16 uint16
			value16, err = s.client.ReadRegister(sensor.Addr, modbus.HOLDING_REGISTER)
			if sensor.Signed {
				value = int64(int16(value16))
			} else {
				value = int64(value16)
			}
		} else if sensor.Words == 2 {
			var value32 uint32
			value32, err = s.client.ReadUint32(sensor.Addr, modbus.HOLDING_REGISTER)
			if sensor.Signed {
				value = int64(int32(value32))
			} else {
				value = int64(value32)
			}
		}
		if err != nil {
			fmt.Println(err)
		}
		sensor.Previous = sensor.Current
		sensor.Current.Value = float64(value) * sensor.Scale
		sensor.Current.Time = now
	}
	return nil
}

func (s *scanner) fixSolarProduction() {
	// the solar production value get reported less frequently than the to-grid
	// value. So we apply a correction offset to keep the solar production
	// value in sync with the to-grid value.
	// Otherwise home-assistant doesn't know where the to-grid power is coming from.

	if s.sensorSolarProduction.Current == s.sensorSolarProduction.Previous {
		// solar production didn't change
		if s.sensorToGrid.Current.Value > s.sensorToGrid.Previous.Value {
			// but to-grid increased, so we need to increase our correction
			fmt.Println("increasing solar production correction")
			s.correctionSolarProduction +=
				(s.sensorToGrid.Current.Value - s.sensorToGrid.Previous.Value)
			fmt.Println("new solar production correction:", s.correctionSolarProduction)
		}
	} else if s.correctionSolarProduction > 0 {
		// solar production changed and we have a correction applied
		if s.sensorSolarProduction.Current.Value >= s.sensorSolarProduction.Previous.Value {
			fmt.Println("reseting solar production correction")
			s.correctionSolarProduction = 0
		} else {
			fmt.Println("reducing solar production correction")
			s.correctionSolarProduction = s.sensorSolarProduction.Previous.Value - s.sensorSolarProduction.Current.Value
			fmt.Println("new solar production correction:", s.correctionSolarProduction)
		}
	}

	s.sensorSolarProduction.Current.Value += s.correctionSolarProduction
}

func (s *scanner) GetMeasurementInfos(filter MeasurementFilter) []Measurement {
	result := make([]Measurement, 0, len(s.sensors))
	for _, sensor := range s.sensors {
		if !filter.MatchName(sensor.Name) {
			continue
		}
		result = append(result, Measurement{
			Name: sensor.Name,
			Unit: sensor.Unit,
		})
	}
	return result
}

func (s *scanner) GetMeasurements(filter MeasurementFilter) []Measurement {
	if s == nil {
		return []Measurement{}
	}

	measurements := make([]Measurement, 0, len(s.sensors))
	for _, sensor := range s.sensors {
		if !filter.MatchName(sensor.Name) {
			continue
		}

		values := []MeasurementValue{}
		if (filter.Previous || filter.Siblings) && !sensor.Previous.Time.IsZero() && filter.MatchTime(sensor.Previous.Time) {
			values = append(values, sensor.Previous)
		}
		if !sensor.Current.Time.IsZero() && filter.MatchTime(sensor.Current.Time) {
			values = append(values, sensor.Current)
		}
		if len(values) == 0 && !sensor.Previous.Time.IsZero() && filter.MatchTime(sensor.Previous.Time) {
			// special case: include previous value if no current value exists (or doesn't match the time frame)
			values = append(values, sensor.Previous)
		}

		measurements = append(measurements, Measurement{
			Name:   sensor.Name,
			Values: values,
			Unit:   sensor.Unit,
		})
	}
	return measurements
}
