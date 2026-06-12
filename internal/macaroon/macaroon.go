package macaroon

import (
	"encoding/base64"
	"fmt"
	"strings"

	macaroon "gopkg.in/macaroon.v2"
)

// caveat keys this verifier understands
const (
	cavPrincipal = "principal"
	cavMethod    = "method"
	cavPath      = "path-prefix"
)

// Request is the context a macaroon is checked against.
type Request struct {
	Principal string
	Method    string
	Path      string
}

// Mint creates a root macaroon bound to key and identified by principal.
func Mint(key []byte, principal string, caveats []string) (*macaroon.Macaroon, error) {
	m, err := macaroon.New(key, []byte(principal), "", macaroon.LatestVersion)
	if err != nil {
		return nil, err
	}
	return addCaveats(m, caveats)
}

// Attenuate returns a copy of m with additional restricting caveats.
func Attenuate(m *macaroon.Macaroon, caveats []string) (*macaroon.Macaroon, error) {
	return addCaveats(m.Clone(), caveats)
}

// Serialize encodes a macaroon for transport over the wire.
func Serialize(m *macaroon.Macaroon) (string, error) {
	b, err := m.MarshalBinary()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// Parse decodes a macaroon received over the wire.
func Parse(s string) (*macaroon.Macaroon, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	m := &macaroon.Macaroon{}
	if err := m.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return m, nil
}

// Verify checks the signature under key and that every caveat holds for req.
func Verify(key []byte, m *macaroon.Macaroon, req Request) error {
	return m.Verify(key, func(caveat string) error {
		return checkCaveat(caveat, req)
	}, nil)
}

func addCaveats(m *macaroon.Macaroon, caveats []string) (*macaroon.Macaroon, error) {
	for _, c := range caveats {
		if err := m.AddFirstPartyCaveat([]byte(c)); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func checkCaveat(caveat string, req Request) error {
	key, val, ok := splitCaveat(caveat)
	if !ok {
		return fmt.Errorf("malformed caveat: %s", caveat)
	}
	switch key {
	case cavPrincipal:
		if val != req.Principal {
			return fmt.Errorf("principal %q does not satisfy %q", req.Principal, val)
		}
	case cavMethod:
		if val != req.Method {
			return fmt.Errorf("method %q not permitted", req.Method)
		}
	case cavPath:
		if !strings.HasPrefix(req.Path, val) {
			return fmt.Errorf("path %q outside prefix %q", req.Path, val)
		}
	default:
		return fmt.Errorf("unknown caveat key %q", key)
	}
	return nil
}

func splitCaveat(c string) (key, val string, ok bool) {
	parts := strings.SplitN(c, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}
