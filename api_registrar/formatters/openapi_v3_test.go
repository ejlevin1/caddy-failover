package formatters

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestOpenAPIv3Formatter_Format(t *testing.T) {
	formatter := &OpenAPIv3Formatter{}

	// Create test specs
	specs := map[string]*CaddyModuleApiSpec{
		"test_api": {
			ID:          "test_api",
			Title:       "Test API",
			Version:     "1.0",
			Description: "A test API",
			Endpoints: []CaddyModuleApiEndpoint{
				{
					Method:      "GET",
					Path:        "/status",
					Summary:     "Get status",
					Description: "Returns the current status",
					Responses: map[int]ResponseDef{
						200: {
							Description: "Success",
							Body: struct {
								Status string `json:"status"`
							}{},
						},
					},
				},
				{
					Method:      "POST",
					Path:        "/items/{id}",
					Summary:     "Create item",
					Description: "Creates a new item",
					PathParams: []Parameter{
						{
							Name:        "id",
							Description: "Item ID",
							Required:    true,
							Type:        "string",
						},
					},
					Request: struct {
						Name  string `json:"name"`
						Value int    `json:"value"`
					}{},
					Responses: map[int]ResponseDef{
						201: {
							Description: "Created",
							Body: struct {
								ID   string `json:"id"`
								Name string `json:"name"`
							}{},
						},
						400: {
							Description: "Bad request",
						},
					},
				},
			},
		},
	}

	// Create test configs
	configs := map[string]*ApiConfig{
		"test_api": {
			Path:    "/api/v1",
			Enabled: true,
		},
	}

	// Format the specs
	result, err := formatter.Format(specs, configs)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Verify the result is an OpenAPISpec
	openapi, ok := result.(*OpenAPISpec)
	if !ok {
		t.Fatal("Result is not an OpenAPISpec")
	}

	// Verify OpenAPI version
	if openapi.OpenAPI != "3.0.3" {
		t.Errorf("Expected OpenAPI version 3.0.3, got %s", openapi.OpenAPI)
	}

	// Verify paths were created correctly
	expectedPaths := []string{"/api/v1/status", "/api/v1/items/{id}"}
	for _, path := range expectedPaths {
		if _, exists := openapi.Paths[path]; !exists {
			t.Errorf("Expected path %s not found", path)
		}
	}

	// Verify GET /api/v1/status
	statusPath := openapi.Paths["/api/v1/status"]
	if statusPath.Get == nil {
		t.Error("GET operation not found for /api/v1/status")
	} else {
		if statusPath.Get.Summary != "Get status" {
			t.Errorf("Expected summary 'Get status', got '%s'", statusPath.Get.Summary)
		}
		if _, exists := statusPath.Get.Responses["200"]; !exists {
			t.Error("200 response not found for GET /api/v1/status")
		}
	}

	// Verify POST /api/v1/items/{id}
	itemsPath := openapi.Paths["/api/v1/items/{id}"]
	if itemsPath.Post == nil {
		t.Error("POST operation not found for /api/v1/items/{id}")
	} else {
		if itemsPath.Post.Summary != "Create item" {
			t.Errorf("Expected summary 'Create item', got '%s'", itemsPath.Post.Summary)
		}
		if len(itemsPath.Post.Parameters) != 1 {
			t.Errorf("Expected 1 parameter, got %d", len(itemsPath.Post.Parameters))
		}
		if itemsPath.Post.RequestBody == nil {
			t.Error("RequestBody not found for POST /api/v1/items/{id}")
		}
		if _, exists := itemsPath.Post.Responses["201"]; !exists {
			t.Error("201 response not found for POST /api/v1/items/{id}")
		}
		if _, exists := itemsPath.Post.Responses["400"]; !exists {
			t.Error("400 response not found for POST /api/v1/items/{id}")
		}
	}
}

func TestOpenAPIv3Formatter_DisabledAPI(t *testing.T) {
	formatter := &OpenAPIv3Formatter{}

	specs := map[string]*CaddyModuleApiSpec{
		"disabled_api": {
			ID:    "disabled_api",
			Title: "Disabled API",
			Endpoints: []CaddyModuleApiEndpoint{
				{
					Method: "GET",
					Path:   "/test",
				},
			},
		},
	}

	configs := map[string]*ApiConfig{
		"disabled_api": {
			Path:    "/api",
			Enabled: false, // Disabled
		},
	}

	result, err := formatter.Format(specs, configs)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	openapi := result.(*OpenAPISpec)

	// Verify disabled API is not included
	if len(openapi.Paths) != 0 {
		t.Error("Expected no paths for disabled API")
	}
}

func TestOpenAPIv3Formatter_Write(t *testing.T) {
	formatter := &OpenAPIv3Formatter{}

	// Create a simple OpenAPI spec
	spec := &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:   "Test API",
			Version: "1.0",
		},
		Paths: map[string]*PathItem{
			"/test": {
				Get: &Operation{
					Summary: "Test endpoint",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
	}

	// Write to buffer
	var buf bytes.Buffer
	err := formatter.Write(&buf, spec)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON output: %v", err)
	}

	// Verify basic structure
	if result["openapi"] != "3.0.3" {
		t.Errorf("Expected openapi version 3.0.3, got %v", result["openapi"])
	}

	info, ok := result["info"].(map[string]interface{})
	if !ok {
		t.Fatal("Info field not found or invalid")
	}
	if info["title"] != "Test API" {
		t.Errorf("Expected title 'Test API', got %v", info["title"])
	}
}

func TestOpenAPIv3Formatter_ContentType(t *testing.T) {
	formatter := &OpenAPIv3Formatter{}

	contentType := formatter.ContentType()
	if contentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got '%s'", contentType)
	}
}

func TestOpenAPIv31Formatter(t *testing.T) {
	formatter := &OpenAPIv31Formatter{}

	specs := map[string]*CaddyModuleApiSpec{
		"test_api": {
			ID:      "test_api",
			Title:   "Test API",
			Version: "1.0",
			Endpoints: []CaddyModuleApiEndpoint{
				{
					Method:  "GET",
					Path:    "/test",
					Summary: "Test endpoint",
					Responses: map[int]ResponseDef{
						200: {Description: "Success"},
					},
				},
			},
		},
	}

	configs := map[string]*ApiConfig{
		"test_api": {
			Path:    "/api",
			Enabled: true,
		},
	}

	result, err := formatter.Format(specs, configs)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	openapi := result.(*OpenAPISpec)

	// Verify OpenAPI 3.1 version
	if openapi.OpenAPI != "3.1.0" {
		t.Errorf("Expected OpenAPI version 3.1.0, got %s", openapi.OpenAPI)
	}
}

func TestGenerateSchema(t *testing.T) {
	formatter := &OpenAPIv3Formatter{}

	// Test struct schema generation
	type TestStruct struct {
		Name     string            `json:"name"`
		Value    int               `json:"value,omitempty"`
		Tags     []string          `json:"tags"`
		Metadata map[string]string `json:"metadata"`
		Hidden   string            `json:"-"`
		private  string            // Should be skipped
	}

	schema := formatter.generateSchema(TestStruct{})

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	// Check properties
	if _, exists := schema.Properties["name"]; !exists {
		t.Error("Property 'name' not found")
	}
	if _, exists := schema.Properties["value"]; !exists {
		t.Error("Property 'value' not found")
	}
	if _, exists := schema.Properties["tags"]; !exists {
		t.Error("Property 'tags' not found")
	}
	if _, exists := schema.Properties["metadata"]; !exists {
		t.Error("Property 'metadata' not found")
	}
	if _, exists := schema.Properties["Hidden"]; exists {
		t.Error("Property 'Hidden' should not be included (json:\"-\")")
	}
	if _, exists := schema.Properties["private"]; exists {
		t.Error("Property 'private' should not be included (unexported)")
	}

	// Check required fields (omitempty fields should not be required)
	requiredMap := make(map[string]bool)
	for _, field := range schema.Required {
		requiredMap[field] = true
	}

	if !requiredMap["name"] {
		t.Error("Field 'name' should be required")
	}
	if requiredMap["value"] {
		t.Error("Field 'value' should not be required (omitempty)")
	}

	// Test array schema
	arraySchema := formatter.generateSchema([]string{})
	if arraySchema.Type != "array" {
		t.Errorf("Expected type 'array', got '%s'", arraySchema.Type)
	}
	if arraySchema.Items == nil {
		t.Error("Array schema should have Items")
	}

	// Test primitive types
	stringSchema := formatter.generateSchema("")
	if stringSchema.Type != "string" {
		t.Errorf("Expected type 'string', got '%s'", stringSchema.Type)
	}

	intSchema := formatter.generateSchema(0)
	if intSchema.Type != "integer" {
		t.Errorf("Expected type 'integer', got '%s'", intSchema.Type)
	}

	boolSchema := formatter.generateSchema(false)
	if boolSchema.Type != "boolean" {
		t.Errorf("Expected type 'boolean', got '%s'", boolSchema.Type)
	}
}

func TestParameterToSchema(t *testing.T) {
	formatter := &OpenAPIv3Formatter{}

	param := Parameter{
		Name:        "test",
		Description: "Test parameter",
		Type:        "string",
		Format:      "email",
		Pattern:     "[a-z]+",
		Enum:        []string{"option1", "option2"},
		Default:     "option1",
		Example:     "option2",
	}

	schema := formatter.parameterToSchema(param)

	if schema.Type != "string" {
		t.Errorf("Expected type 'string', got '%s'", schema.Type)
	}
	if schema.Description != "Test parameter" {
		t.Errorf("Expected description 'Test parameter', got '%s'", schema.Description)
	}
	if schema.Default != "option1" {
		t.Errorf("Expected default 'option1', got %v", schema.Default)
	}
	if schema.Example != "option2" {
		t.Errorf("Expected example 'option2', got %v", schema.Example)
	}
	if len(schema.Enum) != 2 {
		t.Errorf("Expected 2 enum values, got %d", len(schema.Enum))
	}
}
