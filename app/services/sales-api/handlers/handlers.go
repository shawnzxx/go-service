package handlers

import (
	"net/http"
	"os"

	testgrp "github.com/shawnzxx/service/app/services/sales-api/handlers/v1"
	"github.com/shawnzxx/service/business/web/v1/mid"
	"github.com/shawnzxx/service/foundation/web"
	"go.uber.org/zap"
)

// APIMuxConfig contains all the mandatory systems required by handlers.
type APIMuxConfig struct {
	Shutdown chan os.Signal
	Log      *zap.SugaredLogger
}

// APIMux constructs a http.Handler with all application routes defined.
// Handlers design principal, input can be concrete type or interface type, but output return to caller must be a concrete type
// func APIMux(cfg APIMuxConfig) http.Handler {
func APIMux(cfg APIMuxConfig) *web.App {
	app := web.NewApp(cfg.Shutdown, mid.Logger(cfg.Log))

	app.Handle(http.MethodGet, "/test", testgrp.Test)

	return app
}
