package controllers

import (
	"errors"
	"net/http"

	"github.com/volatiletech/abcweb/abcmiddleware"
	"github.com/volatiletech/abcweb/abcrender"
	"go.uber.org/zap"
)

// The list of error types that can be returned by your controllers.
// These can be bound in routes/routes.go to custom error handlers.
// These error types trigger actions in the errors middleware (routes/routes.go)
var (
	ErrUnauthorized = errors.New("not authorized")
	ErrForbidden    = errors.New("access is forbidden")
)

// Root struct exposes useful variables to every controller route handler.
// If you wanted to pass additional objects to each controller route handler
// you can add them here, and then assign them in app/routes.go where the
// instance of controller is created.
type Root struct {
	Log    *zap.Logger
	Render abcrender.Renderer
}

// Main is the controller struct for the main routes (home, about, etc).
// You can add variables to this controller struct to expose them
// to the controller route handlers attached to this controller.
type Main struct {
	Root
}

// Log returns the Request ID scoped logger from the request Context
// and panics if it cannot be found.
// This is a convenience wrapper -- Log() is nicer than abcmiddleware.Log()
func Log(r *http.Request) *zap.Logger {
	return abcmiddleware.Log(r)
}
