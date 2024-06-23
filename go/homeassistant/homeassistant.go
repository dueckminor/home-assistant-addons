package homeassistant

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Api interface {
	GetEntityValue(entityId string) (value string, err error)
}

type api struct {
	token  string
	client *http.Client
}

func NewAPI() Api {
	a := &api{}
	a.token = os.Getenv("SUPERVISOR_TOKEN")
	a.client = http.DefaultClient
	return a
}

func (a *api) GetEntityValue(entityId string) (value string, err error) {
	url := fmt.Sprintf("http://supervisor/core/api/states/%s", entityId)
	req, err := http.NewRequest(http.MethodGet, url, nil)

	req.Header.Add("Authorization","Bearer "+a.token)

	if err != nil {
		return "", err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)

	var msg struct {
		State string
	}

	err = json.Unmarshal(body, &msg)
	if err != nil {
		return "", err
	}

	return msg.State, nil
}
