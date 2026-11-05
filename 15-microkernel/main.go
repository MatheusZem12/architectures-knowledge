// main picks the plugins. Removing a line removes a feature, and the kernel does not change.
// DISABLED_PLUGINS="welcome-coupon,notifications" does the same without rebuilding.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"microkernel/kernel"
	"microkernel/plugins/bigordercoupon"
	"microkernel/plugins/fakepayment"
	"microkernel/plugins/flatshipping"
	"microkernel/plugins/notifications"
	"microkernel/plugins/welcomecoupon"
)

func main() {
	available := []kernel.Plugin{
		welcomecoupon.Plugin{},
		bigordercoupon.Plugin{},
		flatshipping.Plugin{},
		fakepayment.Plugin{},
		&notifications.Plugin{},
	}
	disabled := map[string]bool{}
	for _, name := range strings.Split(os.Getenv("DISABLED_PLUGINS"), ",") {
		disabled[strings.TrimSpace(name)] = true
	}

	core := kernel.New()
	for _, plugin := range available {
		if !disabled[plugin.Name()] {
			core.Load(plugin)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("15-microkernel on port %s, plugins: %s", port, strings.Join(core.Plugins(), ", "))
	log.Fatal(http.ListenAndServe(":"+port, core.Handler()))
}
