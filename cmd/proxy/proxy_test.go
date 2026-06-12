package main

import (
	"net/http/httptest"
	"testing"

	"github.com/jbcallv/agentmesh/internal/macaroon"
	"github.com/jbcallv/agentmesh/internal/policy"
)

const testPolicy = `
permit(principal == Agent::"orchestrator", action == Action::"GET", resource == Resource::"/hello");
forbid(principal == Agent::"worker", action, resource == Resource::"/admin/secret-data");
`

var testKey = []byte("test-key")

func newProxy(t *testing.T) *Proxy {
	engine, err := policy.Load([]byte(testPolicy))
	if err != nil {
		t.Fatal(err)
	}
	return &Proxy{store: NewStore(), engine: engine, key: testKey}
}

func token(t *testing.T, principal string) string {
	m, err := macaroon.Mint(testKey, principal, []string{"principal=" + principal})
	if err != nil {
		t.Fatal(err)
	}
	s, err := macaroon.Serialize(m)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func permitted(p *Proxy, method, path string) bool {
	r := httptest.NewRequest(method, path, nil)
	r.RemoteAddr = "10.0.0.2:5000"
	id, principal := p.identify(r)
	return p.permit(id, principal, r)
}

func TestPermitWhenPolicyAllowsAndTokenValid(t *testing.T) {
	p := newProxy(t)
	p.store.Add(Identity{IP: "10.0.0.2", Principal: "orchestrator", Token: token(t, "orchestrator")})
	if !permitted(p, "GET", "/hello") {
		t.Fatal("orchestrator GET /hello should be permitted")
	}
}

func TestDenyWhenPolicyForbids(t *testing.T) {
	p := newProxy(t)
	p.store.Add(Identity{IP: "10.0.0.2", Principal: "worker", Token: token(t, "worker")})
	if permitted(p, "GET", "/admin/secret-data") {
		t.Fatal("worker GET /admin/secret-data should be denied (confused deputy)")
	}
}

func TestDenyUnknownSource(t *testing.T) {
	p := newProxy(t)
	if permitted(p, "GET", "/hello") {
		t.Fatal("unknown source should fail closed")
	}
}

func TestDenyInvalidToken(t *testing.T) {
	p := newProxy(t)
	p.store.Add(Identity{IP: "10.0.0.2", Principal: "orchestrator", Token: "not-a-macaroon"})
	if permitted(p, "GET", "/hello") {
		t.Fatal("invalid token must fail the intersection even when policy allows")
	}
}
