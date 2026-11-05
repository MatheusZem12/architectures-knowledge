package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// Testing the saga requires simulating the other services as real HTTP servers.
func TestRefusedCouponReleasesTheReservation(t *testing.T) {
	var released atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/reservations":
			_ = json.NewEncoder(w).Encode([]Item{{ProductID: "p", Quantity: 1, UnitPriceCents: 100}})
		case "/releases":
			released.Add(1)
			w.WriteHeader(204)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer remote.Close()

	service := NewService(NewClients(remote.URL, remote.URL, remote.URL, remote.URL))
	_, err := service.Place(PlaceInput{CustomerID: "c", Coupon: "NOSUCHCOUPON", Items: []Line{{"p", 1}}})

	var refusal *RemoteError
	if !errors.As(err, &refusal) || refusal.Code != "invalid_coupon" {
		t.Fatalf("error = %v", err)
	}
	if released.Load() != 1 {
		t.Fatalf("compensation called %d times, want 1", released.Load())
	}
}

func TestCatalogDownBecomesServiceUnavailable(t *testing.T) {
	customers := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) }))
	defer customers.Close()
	down := httptest.NewServer(http.NotFoundHandler())
	down.Close() // nobody answers at this address

	service := NewService(NewClients(down.URL, customers.URL, down.URL, down.URL))
	_, err := service.Place(PlaceInput{CustomerID: "c", Items: []Line{{"p", 1}}})

	if !errors.Is(err, errUnavailable) {
		t.Fatalf("error = %v, want unavailability", err)
	}
}
