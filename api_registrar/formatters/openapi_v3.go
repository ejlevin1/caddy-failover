package formatters

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// OpenAPI 3.0 type definitions
type OpenAPISpec struct {
	OpenAPI    string               `json:"openapi"`
	Info       Info                 `json:"info"`
	Servers    []Server             `json:"servers,omitempty"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`
}

type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

type Server struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty"`
}

type ServerVariable struct {
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

type Operation struct {
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	OperationID string              `json:"operationId,omitempty"`
	Parameters  []ParameterObject   `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type ParameterObject struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // path, query, header, cookie
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Schema      *Schema     `json:"schema"`
	Example     interface{} `json:"example,omitempty"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type MediaType struct {
	Schema  *Schema     `json:"schema,omitempty"`
	Example interface{} `json:"example,omitempty"`
}

type Schema struct {
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Enum        []interface{}      `json:"enum,omitempty"`
	Default     interface{}        `json:"default,omitempty"`
	Example     interface{}        `json:"example,omitempty"`
	Ref         string             `json:"$ref,omitempty"`
}

type Components struct {
	Schemas map[string]*Schema `json:"schemas,omitempty"`
}

// OpenAPIv3Formatter formats API specs as OpenAPI 3.0
type OpenAPIv3Formatter struct {
	ServerURL string // Optional server URL override
}

// Format converts the API specs to OpenAPI 3.0 format
func (f *OpenAPIv3Formatter) Format(specs map[string]*CaddyModuleApiSpec, configs map[string]*ApiConfig) (interface{}, error) {
	// Determine server URL
	serverURL := f.ServerURL
	if serverURL == "" {
		serverURL = "http://localhost"
	}

	openapi := &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       "Caddy Server API",
			Description: "Caddy web server administration and module APIs",
			Version:     "2.0.0",
		},
		Servers: []Server{
			{
				URL:         serverURL,
				Description: "Default server",
			},
		},
		Paths: make(map[string]*PathItem),
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}

	// Process each configured API
	for id, config := range configs {
		if !config.Enabled {
			continue
		}

		spec, exists := specs[id]
		if !exists {
			continue
		}

		// Process endpoints
		for _, endpoint := range spec.Endpoints {
			path := config.Path + endpoint.Path

			// Get or create path item
			pathItem, exists := openapi.Paths[path]
			if !exists {
				pathItem = &PathItem{}
				openapi.Paths[path] = pathItem
			}

			// Create operation
			operation := f.createOperation(endpoint, spec.ID)

			// Assign to correct method
			switch strings.ToUpper(endpoint.Method) {
			case "GET":
				pathItem.Get = operation
			case "POST":
				pathItem.Post = operation
			case "PUT":
				pathItem.Put = operation
			case "PATCH":
				pathItem.Patch = operation
			case "DELETE":
				pathItem.Delete = operation
			}
		}
	}

	return openapi, nil
}

// createOperation creates an OpenAPI operation from an endpoint
func (f *OpenAPIv3Formatter) createOperation(endpoint CaddyModuleApiEndpoint, apiID string) *Operation {
	op := &Operation{
		Summary:     endpoint.Summary,
		Description: endpoint.Description,
		OperationID: f.generateOperationID(apiID, endpoint.Method, endpoint.Path),
		Parameters:  []ParameterObject{},
		Responses:   make(map[string]Response),
	}

	// Add path parameters
	for _, param := range endpoint.PathParams {
		op.Parameters = append(op.Parameters, ParameterObject{
			Name:        param.Name,
			In:          "path",
			Description: param.Description,
			Required:    param.Required,
			Schema:      f.parameterToSchema(param),
			Example:     param.Example,
		})
	}

	// Add query parameters
	for _, param := range endpoint.QueryParams {
		op.Parameters = append(op.Parameters, ParameterObject{
			Name:        param.Name,
			In:          "query",
			Description: param.Description,
			Required:    param.Required,
			Schema:      f.parameterToSchema(param),
			Example:     param.Example,
		})
	}

	// Add header parameters
	for _, param := range endpoint.Headers {
		op.Parameters = append(op.Parameters, ParameterObject{
			Name:        param.Name,
			In:          "header",
			Description: param.Description,
			Required:    param.Required,
			Schema:      f.parameterToSchema(param),
			Example:     param.Example,
		})
	}

	// Add request body if present
	if endpoint.Request != nil {
		op.RequestBody = &RequestBody{
			Description: "Request body",
			Required:    true,
			Content: map[string]MediaType{
				"application/json": {
					Schema: f.generateSchema(endpoint.Request),
				},
			},
		}
	}

	// Add responses
	for statusCode, responseDef := range endpoint.Responses {
		response := Response{
			Description: responseDef.Description,
		}

		if responseDef.Body != nil {
			response.Content = map[string]MediaType{
				"application/json": {
					Schema: f.generateSchema(responseDef.Body),
				},
			}
		}

		op.Responses[fmt.Sprintf("%d", statusCode)] = response
	}

	// Ensure at least one response
	if len(op.Responses) == 0 {
		op.Responses["200"] = Response{
			Description: "Successful response",
		}
	}

	return op
}

// parameterToSchema converts a Parameter to an OpenAPI Schema
func (f *OpenAPIv3Formatter) parameterToSchema(param Parameter) *Schema {
	schema := &Schema{
		Type:        param.Type,
		Format:      param.Format,
		Description: param.Description,
		Default:     param.Default,
		Example:     param.Example,
	}

	if param.Pattern != "" {
		// Note: OpenAPI 3.0 uses "pattern" property
		schema.Format = param.Pattern
	}

	if len(param.Enum) > 0 {
		schema.Enum = make([]interface{}, len(param.Enum))
		for i, v := range param.Enum {
			schema.Enum[i] = v
		}
	}

	return schema
}

// generateSchema generates an OpenAPI schema from a Go type
func (f *OpenAPIv3Formatter) generateSchema(v interface{}) *Schema {
	if v == nil {
		return &Schema{Type: "object"}
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		return f.generateStructSchema(t)
	case reflect.Slice, reflect.Array:
		elemType := t.Elem()
		return &Schema{
			Type:  "array",
			Items: f.generateSchema(reflect.New(elemType).Elem().Interface()),
		}
	case reflect.Map:
		return &Schema{
			Type: "object",
		}
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	default:
		return &Schema{Type: "object"}
	}
}

// generateStructSchema generates a schema for a struct type
func (f *OpenAPIv3Formatter) generateStructSchema(t reflect.Type) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Required:   []string{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		omitempty := false

		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
			for _, part := range parts[1:] {
				if part == "omitempty" {
					omitempty = true
				}
			}
		}

		// Generate schema for field
		fieldSchema := f.generateSchema(reflect.New(field.Type).Elem().Interface())

		// Add description from struct tag if present
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema.Description = desc
		}

		schema.Properties[fieldName] = fieldSchema

		// Add to required if not omitempty
		if !omitempty {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

// generateOperationID generates a unique operation ID
func (f *OpenAPIv3Formatter) generateOperationID(apiID, method, path string) string {
	// Clean up the path to make a valid operation ID
	cleanPath := strings.ReplaceAll(path, "/", "_")
	cleanPath = strings.ReplaceAll(cleanPath, "{", "")
	cleanPath = strings.ReplaceAll(cleanPath, "}", "")
	cleanPath = strings.TrimPrefix(cleanPath, "_")

	return fmt.Sprintf("%s_%s_%s", apiID, strings.ToLower(method), cleanPath)
}

// ContentType returns the HTTP content type for OpenAPI JSON
func (f *OpenAPIv3Formatter) ContentType() string {
	return "application/json"
}

// Write outputs the formatted spec to the writer
func (f *OpenAPIv3Formatter) Write(w io.Writer, spec interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(spec)
}

// OpenAPIv31Formatter formats API specs as OpenAPI 3.1
type OpenAPIv31Formatter struct {
	OpenAPIv3Formatter
}

// Format converts the API specs to OpenAPI 3.1 format
func (f *OpenAPIv31Formatter) Format(specs map[string]*CaddyModuleApiSpec, configs map[string]*ApiConfig) (interface{}, error) {
	// Get the base OpenAPI 3.0 spec
	spec, err := f.OpenAPIv3Formatter.Format(specs, configs)
	if err != nil {
		return nil, err
	}

	// Update version to 3.1
	if openapi, ok := spec.(*OpenAPISpec); ok {
		openapi.OpenAPI = "3.1.0"
	}

	return spec, nil
}
