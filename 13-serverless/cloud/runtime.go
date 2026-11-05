package cloud

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	maxQueueAttempts = 3
	queueRetryDelay  = 20 * time.Millisecond
)

// Request and Response are what a function sees. There is no net/http on this side.
type Request struct {
	PathID string
	Body   []byte
}

type Response struct {
	Status int
	Body   any
}

// Env is everything a function can reach: only managed services, never another function's memory.
type Env struct {
	DB    *KV
	Queue *Queue
}

type Function func(Env, Request) Response

// NewEnv gives a test the same services a deployed function gets.
func NewEnv() Env { return Env{DB: NewKV(), Queue: &Queue{}} }

// Queue emulates a managed queue such as SQS.
type Queue struct {
	mu       sync.RWMutex
	triggers map[string]func([]byte)
}

// Publish hands the message over and returns at once. The consumer runs later, on its own.
func (q *Queue) Publish(topic string, message any) {
	raw, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	q.mu.RLock()
	trigger := q.triggers[topic]
	q.mu.RUnlock()
	if trigger != nil {
		go trigger(raw)
	}
}

type deployment struct {
	name        string
	fn          Function
	mu          sync.Mutex
	idle        []time.Time // when each warm instance last finished
	invocations int
	coldStarts  int
}

// Provider emulates the cloud account: it runs functions, scales them and counts what it would bill.
type Provider struct {
	env         Env
	coldStart   time.Duration
	idleTimeout time.Duration
	mux         *http.ServeMux
	mu          sync.Mutex
	deployments []*deployment
}

func New(coldStart, idleTimeout time.Duration) *Provider {
	env := NewEnv()
	p := &Provider{env: env, coldStart: coldStart, idleTimeout: idleTimeout, mux: http.NewServeMux()}
	p.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { respond(w, 200, map[string]string{"status": "ok"}) })
	p.mux.HandleFunc("GET /__cloud/stats", p.stats)
	return p
}

func (p *Provider) deploy(name string, fn Function) *deployment {
	d := &deployment{name: name, fn: fn}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deployments = append(p.deployments, d)
	return d
}

// invoke runs the function on a warm instance, or starts a new one if none is free. This is the
// whole scaling model: concurrent calls simply get more instances.
func (p *Provider) invoke(d *deployment, req Request) Response {
	d.mu.Lock()
	d.invocations++
	warm := false
	now := time.Now()
	for len(d.idle) > 0 && !warm {
		last := len(d.idle) - 1
		warm = now.Sub(d.idle[last]) < p.idleTimeout
		d.idle = d.idle[:last]
	}
	if !warm {
		d.coldStarts++
	}
	d.mu.Unlock()

	if !warm {
		time.Sleep(p.coldStart)
	}
	response := d.fn(p.env, req)

	d.mu.Lock()
	d.idle = append(d.idle, time.Now())
	d.mu.Unlock()
	return response
}

// Route plays the API gateway: it connects an HTTP route to a function.
func (p *Provider) Route(pattern, name string, fn Function) {
	d := p.deploy(name, fn)
	p.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		response := p.invoke(d, Request{PathID: r.PathValue("id"), Body: body})
		respond(w, response.Status, response.Body)
	})
}

// Trigger connects a queue topic to a function. A status of 500 or more counts as a failure and
// is retried. After the last attempt, a real provider would move it to a dead letter queue.
func (p *Provider) Trigger(topic, name string, fn Function) {
	d := p.deploy(name, fn)
	queue := p.env.Queue
	queue.mu.Lock()
	defer queue.mu.Unlock()
	if queue.triggers == nil {
		queue.triggers = map[string]func([]byte){}
	}
	queue.triggers[topic] = func(message []byte) {
		for attempt := 0; attempt < maxQueueAttempts; attempt++ {
			if p.invoke(d, Request{Body: message}).Status < 500 {
				return
			}
			time.Sleep(queueRetryDelay)
		}
		log.Printf("dead letter: %s failed on %s", name, message)
	}
}

func (p *Provider) Gateway() http.Handler { return p.mux }

type functionStats struct {
	Function    string `json:"function"`
	Invocations int    `json:"invocations"`
	ColdStarts  int    `json:"cold_starts"`
}

// stats shows what the bill is based on: you pay for invocations, not for servers.
func (p *Provider) stats(w http.ResponseWriter, _ *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]functionStats, 0, len(p.deployments))
	for _, d := range p.deployments {
		d.mu.Lock()
		out = append(out, functionStats{d.name, d.invocations, d.coldStarts})
		d.mu.Unlock()
	}
	respond(w, 200, out)
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
