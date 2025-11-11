package network

import (
	"io"
	"net"
	"net/http"
	"path"
	"strings"
)

type HttpToHttps interface {
	io.Closer
	SetChallenge(hostname string, token string, challenge string) error
	SetHandler(hostname string, handler http.Handler) error
}

type tokenAndChallenge struct {
	token     string
	challenge string
}

type httpToHttps struct {
	listener   net.Listener
	challenges map[string]tokenAndChallenge
	handlers   map[string]http.Handler
}

func NewHttpToHttps(network string, address string) (HttpToHttps, error) {
	h := &httpToHttps{
		challenges: make(map[string]tokenAndChallenge),
		handlers:   make(map[string]http.Handler),
	}
	err := h.start(network, address)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *httpToHttps) Close() error {
	if h.listener != nil {
		h.listener.Close()
		h.listener = nil
	}
	return nil
}

func (h *httpToHttps) start(network string, address string) (err error) {
	h.listener, err = net.Listen("tcp", address)
	if err != nil {
		return err
	}

	go func() {
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
			hostname := strings.ToLower(r.Host)
			if handler, ok := h.handlers[hostname]; ok {
				handler.ServeHTTP(w, r)
				return
			}

			target := "https://" + r.Host + r.URL.Path
			if len(r.URL.RawQuery) > 0 {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		})
		(&http.Server{Handler: mux}).Serve(h.listener)
	}()

	return nil
}

func (h httpToHttps) SetChallenge(hostname string, token string, challenge string) error {
	h.challenges[strings.ToLower(hostname)] = tokenAndChallenge{
		token:     strings.ToLower(token),
		challenge: challenge,
	}
	return nil
}

func (h *httpToHttps) SetHandler(hostname string, handler http.Handler) error {
	hostname = strings.ToLower(hostname)
	if handler == nil {
		delete(h.handlers, hostname)
		return nil
	}
	h.handlers[hostname] = handler
	return nil
}
