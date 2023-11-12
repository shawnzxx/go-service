package handlers

import (
	"net/http"
	"os"

	"github.com/dimfeld/httptreemux/v5"
	testgrp "github.com/shawnzxx/service/app/services/sales-api/handlers/v1"
	"go.uber.org/zap"
)

// APIMuxConfig contains all the mandatory systems required by handlers.
type APIMuxConfig struct {
	Shutdown chan os.Signal
	Log      *zap.SugaredLogger
}

// APIMux constructs a http.Handler with all application routes defined.
// H  andlers design principal, input can be concrete type or interface type, but output return to caller must be a concrete type
// func APIMux(cfg APIMuxConfig) http.Handler {
func APIMux(cfg APIMuxConfig) *httptreemux.ContextMux {
	mux := httptreemux.NewContextMux()

	mux.Handle(http.MethodGet, "/test", testgrp.Test)

	return mux
}
