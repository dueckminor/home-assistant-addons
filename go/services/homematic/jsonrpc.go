package homematic

import (
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"net/http"
)

type jsonRPC struct {
	url       string
	username  string
	password  string
	msgId     int
	sessionId string
}

type jsonRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Id      int            `json:"id"`
	Params  map[string]any `json:"params"`
}

type jsonError struct {
	Name    string
	Code    int
	Message string
}

type jsonResponse struct {
	Id      int             `json:"id"`
	Version string          `json:"version"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonError      `json:"error,omitempty"`
}

func (j *jsonRPC) addSession(method string, params map[string]any) map[string]any {
	if method == "Session.login" || j.sessionId == "" {
		return params
	}

	result := map[string]any{
		"_session_id_": j.sessionId,
	}

	maps.Copy(result, params)

	return result
}

func (j *jsonRPC) exec(method string, params map[string]any, result any) (err error) {
	j.msgId = j.msgId + 1

	params = j.addSession(method, params)

	r := jsonRequest{
		JSONRPC: "1.1",
		Method:  method,
		Params:  params,
		Id:      j.msgId,
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}

	resp, err := http.Post(j.url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}

	var jr jsonResponse

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bodyText, &jr)
	if err != nil {
		return err
	}

	if result != nil {
		err = json.Unmarshal(jr.Result, result)
		if err != nil {
			return err
		}
	}

	return err
}

func (j *jsonRPC) Login() (err error) {
	err = j.exec("Session.login", map[string]any{
		"username": j.username,
		"password": j.password,
	}, &j.sessionId)

	// err = j.exec("system.listMethods", nil, nil)

	return err
}

type DeviceInfo struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Interface string `json:"interface"`
	Type      string `json:"type"`
}

func (j *jsonRPC) DeviceListAllDetail() (deviceInfos []DeviceInfo, err error) {
	j.exec("Device.listAllDetail", nil, &deviceInfos)
	return deviceInfos, nil
}
