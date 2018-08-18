// Package web is responsible for serving browser and API requests
package web

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ShoshinNikita/tags-drive/internal/params"
	"github.com/gorilla/mux"
)

var routes = []struct {
	path     string
	methods  string
	handler  http.HandlerFunc
	needAuth bool
}{
	{"/", "GET", index, false}, // index should check is userdata correct itself
	{"/login", "GET", login, false},
	{"/login", "POST", auth, false},
}

// Start starts the server. It has to run in goroutine
//
// Functions stops when stopChan is closed. If there's any error, function will send it into errChan
// After stopping the server function sends http.ErrServerClosed into errChan
func Start(stopChan chan struct{}, errChan chan<- error) {
	router := mux.NewRouter()
	for _, r := range routes {
		var handler http.Handler = r.handler
		if r.needAuth {
			handler = checkAuth(r.handler)
		}
		router.Path(r.path).Methods(r.methods).Handler(handler)
	}

	server := &http.Server{Addr: params.Port, Handler: router}

	go func() {
		if !params.IsTLS {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		} else {
			errChan <- errors.New("TLS isn't available")
		}
	}()

	<-stopChan
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		errChan <- err
	} else {
		errChan <- http.ErrServerClosed
	}
}
