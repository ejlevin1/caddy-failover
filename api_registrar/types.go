package api_registrar

import "github.com/ejlevin1/caddy-failover/api_registrar/formatters"

// Re-export types from formatters package for backward compatibility
type CaddyModuleApiSpec = formatters.CaddyModuleApiSpec
type CaddyModuleApiEndpoint = formatters.CaddyModuleApiEndpoint
type ResponseDef = formatters.ResponseDef
type Parameter = formatters.Parameter
type ApiConfig = formatters.ApiConfig
type ApiSpecFunc = formatters.ApiSpecFunc
