package testgrp

import (
	"encoding/json"
	"net/http"
)

// Test is our test route path handler.
func Test(w http.ResponseWriter, r *http.Request) {

	// Handler's layer principal, handlers function shall only do below 4 things
	// Validate the pass in data from request
	// Call into the business layer pass in request data
	// Return errors to the middleware to handle error in consistent way
	// Handle ok response since handler know what is success response looks like
	status := struct {
		Status string
	}{
		Status: "OK",
	}

	json.NewEncoder(w).Encode(status)
}
