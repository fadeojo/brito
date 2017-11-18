package rendering

import (
	"html/template"

	"github.com/fadeojo/brito/app"
	"github.com/unrolled/render"
	"github.com/volatiletech/abcweb/abcrender"
)

func CustomHelpers(a *app.App) template.FuncMap {
	return template.FuncMap{
		"config": func() interface{} { return a.Config },
	}
}

func New(a *app.App, templatesDir string, manifest map[string]string) abcrender.Renderer {
	appHelpers := []template.FuncMap{
		abcrender.AppHelpers(manifest),
		CustomHelpers(a),
	}

	renderOpts := render.Options{
		Directory:     templatesDir,
		Layout:        "layouts/main",
		Extensions:    []string{".tmpl", ".html"},
		IsDevelopment: a.Config.Server.RenderRecompile,
		Funcs:         appHelpers,
		DisableHTTPErrorRendering: true,
	}

	return abcrender.New(renderOpts, manifest)
}
