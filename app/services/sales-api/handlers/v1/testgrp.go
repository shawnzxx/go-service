package testgrp

import (
	"context"
	"errors"
	"math/rand"
	"net/http"

	v1 "github.com/shawnzxx/service/business/web/v1"
	"github.com/shawnzxx/service/foundation/web"
)

// Test is our test route path handler.
func Test(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

	// Handler's layer principal, handlers function shall only do below 4 things
	// Validate the pass in data from request
	// Call into the business layer pass in request data
	// Return errors to the middleware to handle error in consistent way
	// Handle ok response since handler know what is success response looks like

	if n := rand.Intn(100); n%2 == 0 {
		return v1.NewRequestError(errors.New("TRUSTED ERROR"), http.StatusUnprocessableEntity)
		// return v1.NewRequestError(web.NewShutdownError("shutdown"), http.StatusInternalServerError)
	}

	status := struct {
		Status string
	}{
		Status: "OK",
	}

	return web.Respond(ctx, w, status, http.StatusOK)
}
