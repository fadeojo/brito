package routes

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fadeojo/brito/app"
	"github.com/fadeojo/brito/controllers"
	"github.com/go-chi/chi"
	"github.com/rs/cors"
	"github.com/volatiletech/abcweb/abcmiddleware"
	"github.com/volatiletech/abcweb/abcserver"
	"gopkg.in/olahol/melody.v1"
)

// FileServer sets up a http.FileServer handler to serve
// static files
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
}

// NewRouter creates a new router
func NewRouter(a *app.App, middlewares []abcmiddleware.MiddlewareFunc) *chi.Mux {
	router := chi.NewRouter()
	m := melody.New()

	// Basic CORS
	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	cors := cors.New(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	router.Use(cors.Handler)

	for _, middleware := range middlewares {
		router.Use(middleware)
	}

	// The common state for each route handler
	root := controllers.Root{
		Render: a.Render,
	}

	// 404 route handler
	notFound := abcserver.NewNotFoundHandler(a.AssetsManifest)
	router.NotFound(notFound.Handler(a.Config.Server, a.Render))

	// 405 route handler
	methodNotAllowed := abcserver.NewMethodNotAllowedHandler()
	router.MethodNotAllowed(methodNotAllowed.Handler(a.Render))

	// error middleware handles controller errors
	errMgr := abcmiddleware.NewErrorManager(a.Render)

	errMgr.Add(abcmiddleware.NewError(controllers.ErrUnauthorized, http.StatusUnauthorized, "errors/401", nil))
	errMgr.Add(abcmiddleware.NewError(controllers.ErrForbidden, http.StatusForbidden, "errors/403", nil))

	// Make a pointer to the errMgr.Errors function so it's easier to call
	e := errMgr.Errors

	main := controllers.Main{Root: root}
	router.Get("/", e(main.Home))

	router.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	// Router endpoint for serving reat app in /ui
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "ui")
	fmt.Println("dir")
	fmt.Println(filesDir)
	FileServer(router, "/ui", http.Dir(filesDir))

	return router
}
