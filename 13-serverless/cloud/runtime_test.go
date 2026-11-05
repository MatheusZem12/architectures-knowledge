package cloud

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSecondCallReusesTheWarmInstance(t *testing.T) {
	provider := New(0, time.Minute)
	provider.Route("GET /x", "x", func(Env, Request) Response { return Response{200, "ok"} })
	server := httptest.NewServer(provider.Gateway())
	defer server.Close()

	for range 2 {
		res, err := http.Get(server.URL + "/x")
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
	}

	res, err := http.Get(server.URL + "/__cloud/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `"invocations":2,"cold_starts":1`) {
		t.Fatalf("stats = %s", body)
	}
}

func TestIdleInstanceIsDiscarded(t *testing.T) {
	provider := New(0, 0) // no idle time allowed: every call is a cold start
	d := provider.deploy("x", func(Env, Request) Response { return Response{200, nil} })

	provider.invoke(d, Request{})
	provider.invoke(d, Request{})

	if d.coldStarts != 2 {
		t.Fatalf("cold starts = %d, want 2", d.coldStarts)
	}
}

func TestQueueRetriesAFailedFunction(t *testing.T) {
	provider := New(0, time.Minute)
	var attempts, delivered atomic.Int32
	provider.Trigger("topic", "x", func(Env, Request) Response {
		if attempts.Add(1) < 2 {
			return Response{500, nil}
		}
		delivered.Add(1)
		return Response{200, nil}
	})

	provider.env.Queue.Publish("topic", "message")

	for deadline := time.Now().Add(2 * time.Second); delivered.Load() == 0; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the message was never delivered")
		}
	}
}
