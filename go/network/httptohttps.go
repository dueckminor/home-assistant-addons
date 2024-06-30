package network

import (
	"context"
	"net"
	"net/http"
	"path"
	"strings"
)

type HttpToHttps interface {
	ListenAndServe(ctx context.Context, network string, address string) error
	SetChallenge(hostname string, token string, challenge string) error
}

type tokenAndChallenge struct {
	token     string
	challenge string
}

type httpToHttps struct {
	challenges map[string]tokenAndChallenge
}

func NewHttpToHttps() HttpToHttps {
	return httpToHttps{
		challenges: make(map[string]tokenAndChallenge),
	}
}

func (h httpToHttps) SetChallenge(hostname string, token string, challenge string) error {
	h.challenges[strings.ToLower(hostname)] = tokenAndChallenge{
		token:     strings.ToLower(token),
		challenge: challenge,
	}
	return nil
}

func (h httpToHttps) ListenAndServe(ctx context.Context, network string, address string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/acme-challenge/", func(w http.ResponseWriter, r *http.Request) {
		_, token := path.Split(r.URL.Path)
		hostname := strings.ToLower(r.Host)
		if tAndC, ok := h.challenges[hostname]; ok {
			if token == tAndC.token {
				w.Write([]byte(tAndC.challenge))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		target := "https://" + r.Host + r.URL.Path
		if len(r.URL.RawQuery) > 0 {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	return (&http.Server{Handler: mux}).Serve(listener)
}
