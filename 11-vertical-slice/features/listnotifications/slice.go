// Package listnotifications lists what a customer was told about (R14).
package listnotifications

import (
	"net/http"

	"verticalslice/store"
	"verticalslice/web"
)

func Register(mux *http.ServeMux, db *store.DB) {
	mux.HandleFunc("GET /customers/{id}/notifications", func(w http.ResponseWriter, r *http.Request) {
		var list []store.Notification
		_ = db.Atomic(func() error {
			list = append([]store.Notification{}, db.Notifications[r.PathValue("id")]...)
			return nil
		})
		web.JSON(w, 200, list)
	})
}
