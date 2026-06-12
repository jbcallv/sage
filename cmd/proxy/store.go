package main

import "sync"

type Identity struct {
	IP        string `json:"ip"`
	Principal string `json:"principal"`
	Token     string `json:"token"`
}

type Store struct {
	mu   sync.RWMutex
	byIP map[string]Identity
}

func NewStore() *Store {
	return &Store{byIP: make(map[string]Identity)}
}

func (s *Store) Add(id Identity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byIP[id.IP] = id
}

func (s *Store) Get(ip string) (Identity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byIP[ip]
	return id, ok
}
