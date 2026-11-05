// Package notifications is the largest plugin: it has its own data and its own route.
// The whole feature comes and goes with this one folder.
package notifications

import (
	"net/http"
	"sync"

	"microkernel/kernel"
)

type notification struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

type Plugin struct {
	mu    sync.RWMutex
	items map[string][]notification
}

func (p *Plugin) Name() string { return "notifications" }

func (p *Plugin) Install(registry *kernel.Registry) {
	p.items = map[string][]notification{}
	registry.OnOrderEvent(p.record)
	registry.AddRoute("GET /customers/{id}/notifications", p.list)
}

// R14
func (p *Plugin) record(event kernel.OrderEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items[event.CustomerID] = append(p.items[event.CustomerID], notification{event.CustomerID, event.OrderID, event.Type})
}

func (p *Plugin) list(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	kernel.JSON(w, 200, append([]notification{}, p.items[r.PathValue("id")]...))
}
