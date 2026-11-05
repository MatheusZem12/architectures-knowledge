package store

import (
	"fmt"
	"strings"
	"sync"
)

type Record struct {
	ID    string
	Name  string
	Email string
}

type Store struct {
	mu    sync.Mutex
	items map[string]Record
	seq   int
}

func New() *Store { return &Store{items: map[string]Record{}} }

func (s *Store) InsertIfEmailFree(r Record) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.items {
		if strings.EqualFold(existing.Email, r.Email) {
			return Record{}, false
		}
	}
	s.seq++
	r.ID = fmt.Sprintf("cust-%d", s.seq)
	s.items[r.ID] = r
	return r, true
}

func (s *Store) Find(id string) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.items[id]
	return r, ok
}
