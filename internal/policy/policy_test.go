package policy

import "testing"

const doc = `
permit(principal == Agent::"orchestrator", action == Action::"GET", resource == Resource::"/hello");
permit(principal == Agent::"worker", action == Action::"GET", resource == Resource::"/tasks/list");
forbid(principal == Agent::"worker", action, resource == Resource::"/admin/secret-data");
`

func engine(t *testing.T) *Engine {
	e, err := Load([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestAllowsPermitted(t *testing.T) {
	e := engine(t)
	if !e.Allows("orchestrator", "GET", "/hello") {
		t.Fatal("orchestrator GET /hello should be allowed")
	}
	if !e.Allows("worker", "GET", "/tasks/list") {
		t.Fatal("worker GET /tasks/list should be allowed")
	}
}

func TestDeniesForbidden(t *testing.T) {
	e := engine(t)
	if e.Allows("worker", "GET", "/admin/secret-data") {
		t.Fatal("worker GET /admin/secret-data should be denied")
	}
}

func TestDefaultDeny(t *testing.T) {
	e := engine(t)
	if e.Allows("unknown", "GET", "/hello") {
		t.Fatal("unknown principal should be denied by default")
	}
}

func TestLoadRejectsBadPolicy(t *testing.T) {
	if _, err := Load([]byte("this is not cedar")); err == nil {
		t.Fatal("expected parse error on malformed policy")
	}
}
