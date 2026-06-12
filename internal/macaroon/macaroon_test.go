package macaroon

import "testing"

var key = []byte("test-key")

func TestMintAndVerify(t *testing.T) {
	m, err := Mint(key, "orchestrator", []string{"principal=orchestrator", "method=GET"})
	if err != nil {
		t.Fatal(err)
	}
	req := Request{Principal: "orchestrator", Method: "GET", Path: "/hello"}
	if err := Verify(key, m, req); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestVerifyRejectsViolatedCaveat(t *testing.T) {
	m, _ := Mint(key, "worker", []string{"method=GET"})
	req := Request{Principal: "worker", Method: "POST", Path: "/x"}
	if err := Verify(key, m, req); err == nil {
		t.Fatal("expected POST to be rejected by method=GET caveat")
	}
}

func TestAttenuateOnlyNarrows(t *testing.T) {
	m, _ := Mint(key, "worker", []string{"principal=worker"})
	child, err := Attenuate(m, []string{"path-prefix=/tasks"})
	if err != nil {
		t.Fatal(err)
	}
	// child is valid for /tasks/list
	if err := Verify(key, child, Request{Principal: "worker", Path: "/tasks/list"}); err != nil {
		t.Fatalf("expected /tasks/list valid, got %v", err)
	}
	// but not for /admin — the added caveat narrows authority
	if err := Verify(key, child, Request{Principal: "worker", Path: "/admin"}); err == nil {
		t.Fatal("expected /admin to be rejected by path-prefix caveat")
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	m, _ := Mint(key, "orchestrator", []string{"principal=orchestrator"})
	s, err := Serialize(m)
	if err != nil {
		t.Fatal(err)
	}
	back, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(key, back, Request{Principal: "orchestrator"}); err != nil {
		t.Fatalf("round-tripped macaroon failed verify: %v", err)
	}
}

func TestWrongKeyFails(t *testing.T) {
	m, _ := Mint(key, "orchestrator", []string{"principal=orchestrator"})
	if err := Verify([]byte("other-key"), m, Request{Principal: "orchestrator"}); err == nil {
		t.Fatal("expected verification under wrong key to fail")
	}
}
