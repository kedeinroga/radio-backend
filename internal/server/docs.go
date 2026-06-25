package server

import (
	"net/http"

	"radio-backend/docs" // Generated swagger spec

	"github.com/gin-gonic/gin"
)

// scalarHTML renders the Scalar API Reference UI. The spec is loaded from the
// /docs/openapi.json endpoint exposed below.
const scalarHTML = `<!doctype html>
<html>
  <head>
    <title>Radio Backend API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <div id="app"></div>
    <script id="api-reference" data-url="/docs/openapi.json"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`

// registerDocs mounts the Scalar API reference at /docs. It is intentionally
// skipped in production so the documentation is never exposed there.
func (r *Router) registerDocs() {
	if r.isProduction {
		return
	}

	r.engine.GET("/docs", func(c *gin.Context) {
		// Relax the global CSP for this page so the Scalar bundle (and the
		// assets/fonts it pulls) can load from the CDN. Dev-only route.
		c.Header("Content-Security-Policy", "default-src 'self'; "+
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; "+
			"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://fonts.googleapis.com; "+
			"font-src 'self' data: https://cdn.jsdelivr.net https://fonts.gstatic.com; "+
			"img-src 'self' data: https:; "+
			"connect-src 'self' https://cdn.jsdelivr.net; "+
			"worker-src 'self' blob:")
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(scalarHTML))
	})

	// Serve the generated OpenAPI/Swagger spec consumed by Scalar.
	r.engine.GET("/docs/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(docs.SwaggerInfo.ReadDoc()))
	})
}
