package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleBootstrap records an identity supplied by the Tetragon listener.
func (p *Proxy) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	var id Identity
	if err := json.NewDecoder(r.Body).Decode(&id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	p.store.Add(id)
	log.Printf("BOOT  %-12s ip=%s", id.Principal, id.IP)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "principal": id.Principal})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
