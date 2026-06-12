package main

import (
	"io"
	"net/http"
)

// forward relays the request to its real destination, attaching the agent's credential.
func (p *Proxy) forward(w http.ResponseWriter, r *http.Request, id Identity) {
	req, err := http.NewRequest(r.Method, targetURL(r), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	copyHeaders(req.Header, r.Header)
	req.Header.Set("x-macaroon", id.Token)
	req.Header.Set("x-agent-principal", id.Principal)

	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// targetURL reconstructs the real destination from the intercepted request.
func targetURL(r *http.Request) string {
	return "http://" + r.Host + r.URL.RequestURI()
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}
