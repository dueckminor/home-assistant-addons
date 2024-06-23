package httpbin

import (
	"encoding/json"
	"io"
	"net/http"
)

type Api interface {
	GetExternalIp() (value string, err error)
}

type api struct {
}

func NewAPI() Api {
	a := &api{}
	return a
}

func (a *api) GetExternalIp() (value string, err error) {
	resp, err := http.Get("http://httpbin.org/ip")
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var msg struct {
		Origin string
	}

	err = json.Unmarshal(body, &msg)
	if err != nil {
		return "", err
	}

	return msg.Origin, nil
}
