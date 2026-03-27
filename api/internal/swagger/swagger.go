package swagger

import (
	_ "embed"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:embed openapi.yaml
var spec []byte

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Ashiato API - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/swagger/openapi.yaml",
      dom_id: "#swagger-ui",
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout",
    });
  </script>
</body>
</html>`

// Register adds Swagger UI routes to the Echo instance.
func Register(e *echo.Echo) {
	e.GET("/swagger", func(c echo.Context) error {
		return c.HTML(http.StatusOK, htmlTemplate)
	})
	e.GET("/swagger/openapi.yaml", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "application/yaml", spec)
	})
}
