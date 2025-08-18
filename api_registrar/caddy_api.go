package api_registrar

import "encoding/json"

func init() {
	// Register Caddy Admin API specification
	RegisterApiSpec("caddy_api", getCaddyAdminApiSpec)
}

// CaddyConfig represents the Caddy configuration structure
type CaddyConfig struct {
	Apps map[string]json.RawMessage `json:"apps,omitempty" description:"Caddy applications configuration"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error string `json:"error" description:"Error message"`
}

// UpstreamStatus represents the status of a reverse proxy upstream
type UpstreamStatus struct {
	Address  string `json:"address" description:"Upstream address"`
	Healthy  bool   `json:"healthy" description:"Whether the upstream is healthy"`
	NumFails int    `json:"num_fails" description:"Number of consecutive failures"`
	Fails    int    `json:"fails" description:"Total number of failures"`
}

// PKIInfo represents PKI CA information
type PKIInfo struct {
	ID       string `json:"id" description:"CA identifier"`
	Name     string `json:"name" description:"CA name"`
	RootCert string `json:"root_cert" description:"Root certificate in PEM format"`
	RootKey  string `json:"root_key,omitempty" description:"Root key (if accessible)"`
}

// CertificateChain represents a certificate chain
type CertificateChain struct {
	Certificates []string `json:"certificates" description:"Certificate chain in PEM format"`
}

// getCaddyAdminApiSpec returns the Caddy Admin API specification
func getCaddyAdminApiSpec() *CaddyModuleApiSpec {
	return &CaddyModuleApiSpec{
		ID:          "caddy_api",
		Title:       "Caddy Admin API",
		Version:     "2.0",
		Description: "Caddy server administration API for managing configuration and server state",
		Endpoints: []CaddyModuleApiEndpoint{
			// POST /load
			{
				Method:      "POST",
				Path:        "/load",
				Summary:     "Load configuration",
				Description: "Sets or replaces the active configuration. This is the primary way to configure Caddy.",
				Request:     &CaddyConfig{},
				Responses: map[int]ResponseDef{
					200: {Description: "Configuration loaded successfully"},
					400: {Description: "Invalid configuration", Body: &ErrorResponse{}},
				},
			},
			// POST /stop
			{
				Method:      "POST",
				Path:        "/stop",
				Summary:     "Stop server",
				Description: "Stops the active configuration and exits the process",
				Responses: map[int]ResponseDef{
					200: {Description: "Server stopping"},
				},
			},
			// GET /config/[path]
			{
				Method:      "GET",
				Path:        "/config/{path}",
				Summary:     "Export configuration",
				Description: "Exports the configuration at the named path. If path is omitted, returns the entire configuration.",
				PathParams: []Parameter{
					{
						Name:        "path",
						Description: "Configuration path (e.g., 'apps/http/servers/srv0')",
						Required:    false,
						Type:        "string",
					},
				},
				Responses: map[int]ResponseDef{
					200: {Description: "Configuration object", Body: json.RawMessage{}},
					404: {Description: "Path not found", Body: &ErrorResponse{}},
				},
			},
			// POST /config/[path]
			{
				Method:      "POST",
				Path:        "/config/{path}",
				Summary:     "Set or append configuration",
				Description: "Sets or replaces an object; appends to an array",
				PathParams: []Parameter{
					{
						Name:        "path",
						Description: "Configuration path",
						Required:    false,
						Type:        "string",
					},
				},
				Request: json.RawMessage{},
				Responses: map[int]ResponseDef{
					200: {Description: "Configuration updated"},
					400: {Description: "Invalid configuration", Body: &ErrorResponse{}},
				},
				Headers: []Parameter{
					{
						Name:        "Content-Type",
						Description: "Content type of the configuration",
						Required:    false,
						Type:        "string",
						Default:     "application/json",
					},
				},
			},
			// PUT /config/[path]
			{
				Method:      "PUT",
				Path:        "/config/{path}",
				Summary:     "Create configuration",
				Description: "Creates a new object; inserts into an array at specified position",
				PathParams: []Parameter{
					{
						Name:        "path",
						Description: "Configuration path",
						Required:    true,
						Type:        "string",
					},
				},
				Request: json.RawMessage{},
				Responses: map[int]ResponseDef{
					200: {Description: "Configuration created"},
					201: {Description: "Configuration created"},
					400: {Description: "Invalid configuration", Body: &ErrorResponse{}},
					409: {Description: "Configuration already exists", Body: &ErrorResponse{}},
				},
			},
			// PATCH /config/[path]
			{
				Method:      "PATCH",
				Path:        "/config/{path}",
				Summary:     "Update configuration",
				Description: "Replaces an existing object or array element",
				PathParams: []Parameter{
					{
						Name:        "path",
						Description: "Configuration path",
						Required:    true,
						Type:        "string",
					},
				},
				Request: json.RawMessage{},
				Responses: map[int]ResponseDef{
					200: {Description: "Configuration updated"},
					400: {Description: "Invalid configuration", Body: &ErrorResponse{}},
					404: {Description: "Path not found", Body: &ErrorResponse{}},
				},
			},
			// DELETE /config/[path]
			{
				Method:      "DELETE",
				Path:        "/config/{path}",
				Summary:     "Delete configuration",
				Description: "Deletes the value at the named path",
				PathParams: []Parameter{
					{
						Name:        "path",
						Description: "Configuration path",
						Required:    true,
						Type:        "string",
					},
				},
				Responses: map[int]ResponseDef{
					200: {Description: "Configuration deleted"},
					404: {Description: "Path not found", Body: &ErrorResponse{}},
				},
			},
			// POST /adapt
			{
				Method:      "POST",
				Path:        "/adapt",
				Summary:     "Adapt configuration",
				Description: "Adapts a configuration to JSON without running it. Useful for converting Caddyfile to JSON.",
				Request:     "Caddyfile or other configuration format",
				Responses: map[int]ResponseDef{
					200: {Description: "Adapted configuration", Body: &CaddyConfig{}},
					400: {Description: "Invalid configuration", Body: &ErrorResponse{}},
				},
				Headers: []Parameter{
					{
						Name:        "Content-Type",
						Description: "Format of the input configuration",
						Required:    true,
						Type:        "string",
						Example:     "text/caddyfile",
					},
				},
			},
			// GET /pki/ca/{id}
			{
				Method:      "GET",
				Path:        "/pki/ca/{id}",
				Summary:     "Get PKI CA information",
				Description: "Returns information about a particular PKI app CA",
				PathParams: []Parameter{
					{
						Name:        "id",
						Description: "CA identifier",
						Required:    true,
						Type:        "string",
					},
				},
				Responses: map[int]ResponseDef{
					200: {Description: "CA information", Body: &PKIInfo{}},
					404: {Description: "CA not found", Body: &ErrorResponse{}},
				},
			},
			// GET /pki/ca/{id}/certificates
			{
				Method:      "GET",
				Path:        "/pki/ca/{id}/certificates",
				Summary:     "Get CA certificate chain",
				Description: "Returns the certificate chain of a particular PKI app CA",
				PathParams: []Parameter{
					{
						Name:        "id",
						Description: "CA identifier",
						Required:    true,
						Type:        "string",
					},
				},
				Responses: map[int]ResponseDef{
					200: {Description: "Certificate chain", Body: &CertificateChain{}},
					404: {Description: "CA not found", Body: &ErrorResponse{}},
				},
			},
			// GET /reverse_proxy/upstreams
			{
				Method:      "GET",
				Path:        "/reverse_proxy/upstreams",
				Summary:     "Get upstream status",
				Description: "Returns the current status of the configured proxy upstreams",
				Responses: map[int]ResponseDef{
					200: {Description: "Upstream status list", Body: []UpstreamStatus{}},
				},
			},
		},
	}
}
