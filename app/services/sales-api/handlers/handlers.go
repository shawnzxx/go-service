package handlers

import (
	"github.com/jmoiron/sqlx"
	"net/http"
	"os"

	testgrp "github.com/shawnzxx/service/app/services/sales-api/handlers/v1"
	"github.com/shawnzxx/service/business/web/auth"
	"github.com/shawnzxx/service/business/web/v1/mid"
	"github.com/shawnzxx/service/foundation/web"
	"go.uber.org/zap"
)

// APIMuxConfig contains all the mandatory systems required by handlers.
type APIMuxConfig struct {
	Shutdown chan os.Signal
	Log      *zap.SugaredLogger
	Auth     *auth.Auth
	DB       *sqlx.DB
}

// APIMux constructs a http.Handler with all application routes defined.
// Handlers design principal, input can be concrete type or interface type, but output return to caller must be a concrete type
// func APIMux(cfg APIMuxConfig) http.Handler {
func APIMux(cfg APIMuxConfig) *web.App {
	app := web.NewApp(cfg.Shutdown, mid.Logger(cfg.Log), mid.Errors(cfg.Log), mid.Metrics(), mid.Panics())

	app.Handle(http.MethodGet, "/test", testgrp.Test)
	app.Handle(http.MethodGet, "/test/auth", testgrp.Test, mid.Authenticate(cfg.Auth), mid.Authorize(cfg.Auth, auth.RuleAdminOnly))

	return app
}
