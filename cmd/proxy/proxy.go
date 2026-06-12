package main

import (
	"log"
	"net"
	"net/http"

	"github.com/jbcallv/agentmesh/internal/macaroon"
	"github.com/jbcallv/agentmesh/internal/policy"
)

type Proxy struct {
	store  *Store
	engine *policy.Engine
	key    []byte
	client *http.Client
}

// handleRequest enforces policy on an intercepted request, then forwards it.
func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	id, principal := p.identify(r)
	policyOK, tokenOK := p.evaluate(id, principal, r)
	allow := policyOK && tokenOK
	logDecision(r, principal, policyOK, tokenOK, allow)
	if !allow {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "forbidden", "principal": principal, "resource": r.URL.Path,
		})
		return
	}
	p.forward(w, r, id)
}

// identify resolves the calling container's identity by source IP.
func (p *Proxy) identify(r *http.Request) (Identity, string) {
	if id, ok := p.store.Get(clientIP(r)); ok {
		return id, id.Principal
	}
	return Identity{}, "unknown"
}

// evaluate runs both halves of the intersection and reports each for logging.
func (p *Proxy) evaluate(id Identity, principal string, r *http.Request) (policyOK, tokenOK bool) {
	policyOK = p.engine.Allows(principal, r.Method, r.URL.Path)
	tokenOK = p.verify(id, principal, r)
	return
}

// permit is the intersection: the local policy must allow AND the macaroon must verify.
func (p *Proxy) permit(id Identity, principal string, r *http.Request) bool {
	policyOK, tokenOK := p.evaluate(id, principal, r)
	return policyOK && tokenOK
}

// logDecision narrates one enforcement decision in a single line.
func logDecision(r *http.Request, principal string, policyOK, tokenOK, allow bool) {
	verdict := "DENY "
	if allow {
		verdict = "ALLOW"
	}
	log.Printf("%s %-12s %s %s%s  token=%s policy=%s",
		verdict, principal, r.Method, r.Host, r.URL.Path, mark(tokenOK), mark(policyOK))
}

func mark(ok bool) string {
	if ok {
		return "ok"
	}
	return "NO"
}

func (p *Proxy) verify(id Identity, principal string, r *http.Request) bool {
	m, err := macaroon.Parse(id.Token)
	if err != nil {
		return false
	}
	req := macaroon.Request{Principal: principal, Method: r.Method, Path: r.URL.Path}
	return macaroon.Verify(p.key, m, req) == nil
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
