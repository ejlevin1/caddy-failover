package caddyfailover

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/ejlevin1/caddy-failover/api_registrar"
	"github.com/ejlevin1/caddy-failover/failover"
)

func init() {
	caddy.RegisterModule(&failover.FailoverProxy{})
	caddy.RegisterModule(&failover.FailoverStatusHandler{})
	httpcaddyfile.RegisterHandlerDirective("failover_proxy", failover.ParseFailoverProxy)
	httpcaddyfile.RegisterHandlerDirective("failover_status", failover.ParseFailoverStatus)

	// Register failover API specification
	api_registrar.RegisterApiSpec("failover_api", failover.GetFailoverApiSpec)
}

// Export types for external packages
type FailoverProxy = failover.FailoverProxy
type FailoverStatusHandler = failover.FailoverStatusHandler
