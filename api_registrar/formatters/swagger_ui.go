package formatters

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SwaggerUIFormatter serves Swagger UI HTML page
type SwaggerUIFormatter struct {
	SpecURL string // URL to the OpenAPI spec endpoint
}

// Format returns HTML content for Swagger UI
func (f *SwaggerUIFormatter) Format(specs map[string]*CaddyModuleApiSpec, configs map[string]*ApiConfig) (interface{}, error) {
	// Generate the HTML page that includes Swagger UI
	html := f.generateSwaggerUIHTML()
	return html, nil
}

// ContentType returns text/html for the UI
func (f *SwaggerUIFormatter) ContentType() string {
	return "text/html; charset=utf-8"
}

// Write outputs the HTML to the writer
func (f *SwaggerUIFormatter) Write(w io.Writer, spec interface{}) error {
	html, ok := spec.(string)
	if !ok {
		return fmt.Errorf("expected string HTML, got %T", spec)
	}
	_, err := w.Write([]byte(html))
	return err
}

// generateSwaggerUIHTML creates the HTML page with embedded Swagger UI
func (f *SwaggerUIFormatter) generateSwaggerUIHTML() string {
	// Default spec URL if not provided
	specURL := f.SpecURL
	if specURL == "" {
		specURL = "./openapi.json"
	}

	// Ensure the URL is properly escaped for JavaScript
	escapedURL, _ := json.Marshal(specURL)

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>API Documentation - Swagger UI</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui.css">
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin: 0;
            background: #fafafa;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: ` + string(escapedURL) + `,
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                validatorUrl: null,
                tryItOutEnabled: true,
                supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
                onComplete: function() {
                    console.log("Swagger UI loaded successfully");
                }
            });
        };
    </script>
</body>
</html>`
}

// RedocUIFormatter serves Redoc UI HTML page
type RedocUIFormatter struct {
	SpecURL string // URL to the OpenAPI spec endpoint
}

// Format returns HTML content for Redoc UI
func (f *RedocUIFormatter) Format(specs map[string]*CaddyModuleApiSpec, configs map[string]*ApiConfig) (interface{}, error) {
	html := f.generateRedocUIHTML()
	return html, nil
}

// ContentType returns text/html for the UI
func (f *RedocUIFormatter) ContentType() string {
	return "text/html; charset=utf-8"
}

// Write outputs the HTML to the writer
func (f *RedocUIFormatter) Write(w io.Writer, spec interface{}) error {
	html, ok := spec.(string)
	if !ok {
		return fmt.Errorf("expected string HTML, got %T", spec)
	}
	_, err := w.Write([]byte(html))
	return err
}

// generateRedocUIHTML creates the HTML page with embedded Redoc UI
func (f *RedocUIFormatter) generateRedocUIHTML() string {
	// Default spec URL if not provided
	specURL := f.SpecURL
	if specURL == "" {
		specURL = "./openapi.json"
	}

	// For HTML attributes, we just need to escape quotes, not the entire URL
	// Redoc expects a normal URL, not a query-encoded one
	escapedURL := strings.ReplaceAll(specURL, "'", "&#39;")

	return `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation - Redoc</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
    <redoc spec-url='` + escapedURL + `'></redoc>
    <script src="https://cdn.jsdelivr.net/npm/redoc@2.1.3/bundles/redoc.standalone.js"></script>
</body>
</html>`
}

// UpdateGetFormatter updates the GetFormatter function to include UI formatters
func GetFormatterWithUI(format string, currentPath string) Formatter {
	// Extract the base path for relative spec URLs
	basePath := strings.TrimSuffix(currentPath, "/")
	if idx := strings.LastIndex(basePath, "/"); idx >= 0 {
		basePath = basePath[:idx]
	}

	switch format {
	case "swagger-ui", "swaggerui":
		return &SwaggerUIFormatter{
			SpecURL: basePath + "/openapi.json",
		}
	case "redoc", "redoc-ui":
		return &RedocUIFormatter{
			SpecURL: basePath + "/openapi.json",
		}
	case "openapi-v3.0", "openapi-3.0", "openapi":
		return &OpenAPIv3Formatter{}
	case "openapi-v3.1", "openapi-3.1":
		return &OpenAPIv31Formatter{}
	default:
		// Default to OpenAPI 3.0
		return &OpenAPIv3Formatter{}
	}
}
