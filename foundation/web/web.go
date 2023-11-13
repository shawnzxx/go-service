// Package web contains a small web framework extension.
package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dimfeld/httptreemux/v5"
	"github.com/google/uuid"
)

// A Handler is a type that handles a http request within our own little mini framework.
// original Handler in Mux is
// type Handler func(w http.ResponseWriter, r *http.Request)
// we add input: ctx context.Context and output: error
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// App is the entrypoint into our application and what configures our context
// object for each of our http handlers. Feel free to add any configuration
// data/logic on this App struct.
type App struct {
	// this is embeded in App struct (filed without name), so App can use anything from ContextMux
	*httptreemux.ContextMux
	shutdown chan os.Signal
	mw       []Middleware
}

// NewApp creates an App value that handle a set of routes for the application.
// input can pass in 0~many middleware
func NewApp(shutdown chan os.Signal, mw ...Middleware) *App {
	return &App{
		ContextMux: httptreemux.NewContextMux(),
		shutdown:   shutdown,
		mw:         mw,
	}
}

// Handle sets a handler function for a given HTTP method and path pair
// to the application server mux
// input can pass in 0~many middleware
func (a *App) Handle(method string, path string, handler Handler, mw ...Middleware) {
	// handler it's own middleware, depends on whether we inject any MW at handler, like auth middleware not all handlers may need it
	handler = wrapMiddleware(mw, handler)
	// app layer middleware need to execute for all handlers, like logger middleware
	handler = wrapMiddleware(a.mw, handler)

	// we wrpped our own logic into original Handler function
	h := func(w http.ResponseWriter, r *http.Request) {

		// We can add ANY CODE I WANT before

		v := Values{
			TraceID: uuid.NewString(),
			Now:     time.Now().UTC(),
		}
		ctx := context.WithValue(r.Context(), key, &v)

		if err := handler(ctx, w, r); err != nil {
			fmt.Println(err)
			return
		}

		// We can add ANY CODE I WANT after
	}

	a.ContextMux.Handle(method, path, h)
}
