// Package cloud is a local emulator of what a provider sells: a function runtime, an API gateway,
// a key-value database and a queue. In production, this whole folder would not exist in your code.
package cloud

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// KV emulates a managed database such as DynamoDB. Values are stored as JSON, so no function can
// keep a pointer to shared memory: reading always returns a copy, as it would over the network.
type KV struct {
	mu   sync.Mutex
	data map[string][]byte
	seq  int
}

func NewKV() *KV { return &KV{data: map[string][]byte{}} }

// Tx groups reads and writes into one all-or-nothing transaction.
type Tx struct {
	kv     *KV
	writes map[string][]byte
}

// Transact commits only if fn returns no error.
func (k *KV) Transact(fn func(*Tx) error) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	tx := &Tx{kv: k, writes: map[string][]byte{}}
	if err := fn(tx); err != nil {
		return err
	}
	for key, value := range tx.writes {
		k.data[key] = value
	}
	return nil
}

func (t *Tx) Get(key string, into any) bool {
	raw, ok := t.writes[key]
	if !ok {
		raw, ok = t.kv.data[key]
	}
	if !ok {
		return false
	}
	if err := json.Unmarshal(raw, into); err != nil {
		panic(err)
	}
	return true
}

func (t *Tx) Put(key string, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	t.writes[key] = raw
}

func (t *Tx) NextID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, t.Sequence())
}

// Sequence returns a number that only grows, useful to keep keys in order.
func (t *Tx) Sequence() int {
	t.kv.seq++
	return t.kv.seq
}

// Scan reads every stored value whose key starts with prefix, ordered by key. In a real key-value
// database, this is a query on a partition: fine for one customer, expensive across a whole table.
func (t *Tx) Scan(prefix string) []json.RawMessage {
	keys := []string{}
	for key := range t.kv.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]json.RawMessage, len(keys))
	for i, key := range keys {
		out[i] = t.kv.data[key]
	}
	return out
}
