package homematic

import (
	"bytes"
	"encoding/json"
	"sync"
)

type Device interface {
	Name() string
	SetName(name string)
	Address() string
	Type() string
	GetValue(valueName string) (result any, err error)
	SetValue(valueName string, value any) (err error)
	SetValueIfChanged(valueName string, value any) (changed bool, err error)
	SubDevices() []Device
	GetSubDevice(subdeviceType string) (subdevice Device, err error)
	GetMasterDescription() (paramsetDescription ParamsetDescription, err error)
	GetValueDescription() (paramsetDescription ParamsetDescription, err error)
	GetValues() (value map[string]any, err error)
}

type deviceInt interface {
	Device
	setSubdevice(subdevice Device)
	putValue(valueName string, value any)
}

type deviceImpl struct {
	ccuc       *CcuClientImpl
	name       string
	deviceDesc DeviceDescription
	mutex      sync.RWMutex
	subdevices map[string]Device
	values     map[string]any
}

func newDevice(ccuc *CcuClientImpl, deviceDesc DeviceDescription) (dev *deviceImpl) {
	dev = new(deviceImpl)
	dev.ccuc = ccuc
	dev.name = deviceDesc.Address
	dev.deviceDesc = deviceDesc
	dev.initMaps()
	return dev
}

func (dev *deviceImpl) Name() string {
	return dev.name
}

func (dev *deviceImpl) SetName(name string) {
	dev.name = name
}

func (dev *deviceImpl) Address() string {
	return dev.deviceDesc.Address
}

func (dev *deviceImpl) Type() string {
	return dev.deviceDesc.Type
}

func (dev *deviceImpl) initMaps() {
	dev.subdevices = make(map[string]Device)
	dev.values = make(map[string]any)
}

func (dev *deviceImpl) getValueFromCache(valueName string) (result any, found bool) {
	dev.mutex.RLock()
	defer dev.mutex.RUnlock()
	if result, found = dev.values[valueName]; found {
		return result, found
	}
	return nil, false
}

func (dev *deviceImpl) GetValue(valueName string) (result any, err error) {
	if result, ok := dev.getValueFromCache(valueName); ok {
		return result, nil
	}

	result, err = dev.ccuc.GetValue(dev.deviceDesc.Address, valueName)
	if err == nil {
		dev.mutex.Lock()
		defer dev.mutex.Unlock()
		dev.values[valueName] = result
	}
	return result, err
}

func (dev *deviceImpl) GetSubDevice(subdeviceType string) (subdevice Device, err error) {
	dev.mutex.RLock()
	defer dev.mutex.RUnlock()
	if subdevice, ok := dev.subdevices[subdeviceType]; ok {
		return subdevice, nil
	}
	return nil, nil
}

func (dev *deviceImpl) SubDevices() (subdevices []Device) {
	dev.mutex.RLock()
	defer dev.mutex.RUnlock()
	for _, subdevice := range dev.subdevices {
		subdevices = append(subdevices, subdevice)
	}
	return subdevices
}

func (dev *deviceImpl) setSubdevice(subdevice Device) {
	dev.mutex.Lock()
	defer dev.mutex.Unlock()
	dev.subdevices[subdevice.Type()] = subdevice
}

func (dev *deviceImpl) putValueToCache(valueName string, value any) (changed bool) {
	dev.mutex.Lock()
	defer dev.mutex.Unlock()
	if oldValue, ok := dev.values[valueName]; !ok || oldValue != value {
		changed = true
	}
	dev.values[valueName] = value
	return changed
}

func (dev *deviceImpl) putValue(valueName string, value any) {
	dev.putValueToCache(valueName, value)
	for _, callback := range dev.ccuc.callbacks {
		callback(dev, valueName, value)
	}
}

func (dev *deviceImpl) SetValue(valueName string, value any) (err error) {
	err = dev.ccuc.SetValue(dev.deviceDesc.Address, valueName, value)
	if err == nil {
		dev.putValue(valueName, value)
	}
	return err
}

func equals(a, b any) bool {
	da, err := json.Marshal(a)
	if err != nil {
		return false
	}
	db, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(da, db)
}

func (dev *deviceImpl) SetValueIfChanged(valueName string, value any) (changed bool, err error) {
	if oldValue, ok := dev.getValueFromCache(valueName); !ok || !equals(oldValue, value) {
		err = dev.ccuc.SetValue(dev.deviceDesc.Address, valueName, value)
		if err == nil {
			dev.putValueToCache(valueName, value)
		}
		return true, err
	}
	return false, nil
}

func (dev *deviceImpl) GetMasterDescription() (paramsetDescription ParamsetDescription, err error) {
	return dev.ccuc.GetMasterDescription(dev.deviceDesc.Address)
}

func (dev *deviceImpl) GetValueDescription() (paramsetDescription ParamsetDescription, err error) {
	return dev.ccuc.GetValueDescription(dev.deviceDesc.Address)
}

func (dev *deviceImpl) GetValues() (values map[string]any, err error) {
	values, err = dev.ccuc.GetParamset(dev.deviceDesc.Address, "VALUES")
	if err == nil {
		for valueName, value := range values {
			dev.putValue(valueName, value)
		}
	}
	return values, err
}
