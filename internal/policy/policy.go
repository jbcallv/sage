package policy

import "github.com/cedar-policy/cedar-go"

// Engine evaluates a node's local Cedar policy.
type Engine struct {
	policies *cedar.PolicySet
}

// Load parses a Cedar policy document.
func Load(document []byte) (*Engine, error) {
	ps, err := cedar.NewPolicySetFromBytes("agents.cedar", document)
	if err != nil {
		return nil, err
	}
	return &Engine{policies: ps}, nil
}

// Allows reports whether the local policy permits principal to take action on resource.
func (e *Engine) Allows(principal, action, resource string) bool {
	req := cedar.Request{
		Principal: cedar.NewEntityUID("Agent", cedar.String(principal)),
		Action:    cedar.NewEntityUID("Action", cedar.String(action)),
		Resource:  cedar.NewEntityUID("Resource", cedar.String(resource)),
		Context:   cedar.NewRecord(cedar.RecordMap{}),
	}
	decision, _ := cedar.Authorize(e.policies, nil, req)
	return decision == cedar.Allow
}
