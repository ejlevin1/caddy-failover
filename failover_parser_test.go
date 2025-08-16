package caddyfailover

import (
	"testing"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func TestUnmarshalCaddyfileParser(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, *FailoverProxy)
	}{
		{
			name:  "basic configuration",
			input: `failover_proxy http://localhost:8080 http://localhost:8081`,
			validate: func(t *testing.T, fp *FailoverProxy) {
				if len(fp.Upstreams) != 2 {
					t.Errorf("expected 2 upstreams, got %d", len(fp.Upstreams))
				}
				if fp.Upstreams[0] != "http://localhost:8080" {
					t.Errorf("expected first upstream to be http://localhost:8080, got %s", fp.Upstreams[0])
				}
			},
		},
		{
			name: "with fail_duration",
			input: `failover_proxy http://localhost:8080 {
				fail_duration 60s
			}`,
			validate: func(t *testing.T, fp *FailoverProxy) {
				// 60 seconds in nanoseconds
				if fp.FailDuration != 60000000000 {
					t.Errorf("expected fail_duration to be 60s, got %v", fp.FailDuration)
				}
			},
		},
		{
			name: "with insecure_skip_verify",
			input: `failover_proxy https://localhost:8443 {
				insecure_skip_verify
			}`,
			validate: func(t *testing.T, fp *FailoverProxy) {
				if !fp.InsecureSkipVerify {
					t.Error("expected insecure_skip_verify to be true")
				}
			},
		},
		{
			name:    "no upstreams error",
			input:   `failover_proxy`,
			wantErr: true,
		},
		{
			name: "invalid fail_duration",
			input: `failover_proxy http://localhost:8080 {
				fail_duration invalid
			}`,
			wantErr: true,
		},
		{
			name: "unknown directive",
			input: `failover_proxy http://localhost:8080 {
				unknown_directive value
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := caddyfile.NewTestDispenser(tt.input)
			fp := &FailoverProxy{}
			err := fp.UnmarshalCaddyfile(d)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, fp)
			}
		})
	}
}

func TestUnmarshalCaddyfile_MultipleBlocks(t *testing.T) {
	// Test that a single block with multiple directives works
	input := `failover_proxy http://localhost:8080 http://localhost:8081 {
		fail_duration 30s
		insecure_skip_verify
	}`

	d := caddyfile.NewTestDispenser(input)
	fp := &FailoverProxy{}

	err := fp.UnmarshalCaddyfile(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check upstreams
	if len(fp.Upstreams) != 2 {
		t.Errorf("expected 2 upstreams, got %d", len(fp.Upstreams))
	}

	// Check fail_duration
	if fp.FailDuration != 30000000000 {
		t.Errorf("expected fail_duration to be 30s, got %v", fp.FailDuration)
	}

	// Check insecure_skip_verify
	if !fp.InsecureSkipVerify {
		t.Error("expected insecure_skip_verify to be true")
	}
}
