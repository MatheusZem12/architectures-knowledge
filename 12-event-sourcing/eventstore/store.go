// Package eventstore is an append-only log. There is no update and no delete: only Append.
//
// Each stream groups the events of one thing, such as one product or one order.
package eventstore

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"sync"

	"eventsourcing/events"
)

// Record is an event in its place in history.
type Record struct {
	Seq    int             `json:"seq"`
	Stream string          `json:"stream"`
	Type   string          `json:"type"`
	Data   json.RawMessage `json:"data"`
	Event  any             `json:"-"`
}

// Entry is an event about to be appended to a stream.
type Entry struct {
	Stream string
	Event  any
}

type Store struct {
	mu          sync.RWMutex
	records     []Record
	streams     map[string][]int
	subscribers []func(Record)
	file        *os.File
}

// Open replays the log from path, if there is one. With an empty path, it stays in memory.
func Open(path string) (*Store, error) {
	s := &Store{streams: map[string][]int{}}
	if path == "" {
		return s, nil
	}
	existing, err := os.Open(path)
	if err == nil {
		defer existing.Close()
		scanner := bufio.NewScanner(existing)
		scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
		for scanner.Scan() {
			var record Record
			if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
				return nil, err
			}
			if record.Event, err = events.Decode(record.Type, record.Data); err != nil {
				return nil, err
			}
			s.add(record)
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	s.file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	return s, err
}

func (s *Store) add(record Record) {
	s.streams[record.Stream] = append(s.streams[record.Stream], len(s.records))
	s.records = append(s.records, record)
}

// Append records all entries together, then tells the subscribers.
func (s *Store) Append(entries ...Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	added := make([]Record, 0, len(entries))
	for i, entry := range entries {
		data, err := json.Marshal(entry.Event)
		if err != nil {
			return err
		}
		added = append(added, Record{Seq: len(s.records) + i + 1, Stream: entry.Stream, Type: events.Name(entry.Event), Data: data, Event: entry.Event})
	}
	if s.file != nil {
		var lines []byte
		for _, record := range added {
			line, err := json.Marshal(record)
			if err != nil {
				return err
			}
			lines = append(append(lines, line...), '\n')
		}
		if _, err := s.file.Write(lines); err != nil {
			return err
		}
	}
	for _, record := range added {
		s.add(record)
		for _, subscriber := range s.subscribers {
			subscriber(record)
		}
	}
	return nil
}

// Load returns a stream's events, in order: the raw material to rebuild one thing's state.
func (s *Store) Load(stream string) []any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]any, len(s.streams[stream]))
	for i, position := range s.streams[stream] {
		out[i] = s.records[position].Event
	}
	return out
}

// History returns a stream's records, with their position in the log.
func (s *Store) History(stream string) []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Record, len(s.streams[stream]))
	for i, position := range s.streams[stream] {
		out[i] = s.records[position]
	}
	return out
}

// Subscribe first replays the whole past, then delivers what comes next. That is how a brand new
// read model can be created at any time: it just reads the log from the beginning.
func (s *Store) Subscribe(subscriber func(Record)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, record := range s.records {
		subscriber(record)
	}
	s.subscribers = append(s.subscribers, subscriber)
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}
