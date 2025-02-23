package homematic

import (
	"net/url"
	"strings"
)

type CcuMultiClientImpl struct {
	jsonRPC *jsonRPC
	clients []*CcuClientImpl
}

func NewCcuClient(uri string, username string, password string) (ccuc CcuClient, err error) {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	parsedURI.User = url.UserPassword(username, password)

	// port 2001: HM
	// port 2010: HM-IP

	port := parsedURI.Port()
	if len(port) > 0 {
		return newCcuClient(uri)
	}

	host := parsedURI.Host
	parsedURI.Host = host + ":2001"

	client, err := newCcuClient(parsedURI.String())
	if err != nil {
		return nil, err
	}

	ccumc := &CcuMultiClientImpl{
		jsonRPC: &jsonRPC{
			url:      uri + "/api/homematic.cgi",
			username: username,
			password: password,
		},
	}
	ccumc.clients = append(ccumc.clients, client)

	parsedURI.Host = host + ":2010"
	client, err = newCcuClient(parsedURI.String())
	if err != nil {
		return nil, err
	}

	ccumc.clients = append(ccumc.clients, client)
	return ccumc, err
}

func (ccumc *CcuMultiClientImpl) GetVersion() (version string, err error) {
	return ccumc.clients[0].GetVersion()
}
func (ccumc *CcuMultiClientImpl) SetCallback(cb CcuCallback) {
	for _, ccuc := range ccumc.clients {
		ccuc.SetCallback(cb)
	}
}
func (ccumc *CcuMultiClientImpl) StartCallbackHandler() error {
	for _, ccuc := range ccumc.clients {
		err := ccuc.StartCallbackHandler()
		if err != nil {
			return err
		}
	}
	return nil
}

func (ccumc *CcuMultiClientImpl) GetOwnIP() string {
	return ccumc.clients[0].GetOwnIP()
}

func (ccumc *CcuMultiClientImpl) GetDevices() (devices []Device, err error) {
	err = ccumc.jsonRPC.Login()
	if err != nil {
		return nil, err
	}

	deviceInfos, err := ccumc.jsonRPC.DeviceListAllDetail()
	if err != nil {
		return nil, err
	}

	for _, ccuc := range ccumc.clients {
		devs, err := ccuc.GetDevices()
		if err != nil {
			return nil, err
		}
		devices = append(devices, devs...)
	}

	for _, dev := range devices {
		addr := dev.Address()
		name := ""
		for _, deviceInfo := range deviceInfos {
			if deviceInfo.Address == addr || strings.HasPrefix(addr, deviceInfo.Address+":") {
				name = deviceInfo.Name
				break
			}
		}
		if name == "" {
			name = "unknown"
		}
		dev.SetName(name)
	}

	return devices, nil
}

func (ccumc *CcuMultiClientImpl) GetDevice(addr string) (device Device, err error) {
	for _, ccuc := range ccumc.clients {
		dev, _ := ccuc.GetDevice(addr)
		if dev != nil {
			return dev, nil
		}
	}
	return nil, nil
}
