package main

import "testing"

// All routing knowledge lives in the bus, so a typo here would break the whole system.
func TestEveryPublicRouteHasAProvider(t *testing.T) {
	for _, route := range restRoutes {
		if providers[route.operation] == "" {
			t.Errorf("%s points to %s, which has no provider", route.pattern, route.operation)
		}
	}
}
