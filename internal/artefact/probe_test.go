package artefact_test

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
)

// TestResolveProbeHost is ADR-0029's three steps walked at invocation, and the
// four answers they reach. Three of them are declines and each declines
// differently, which is the whole reason the answer is a value rather than a
// host and a boolean (§3, §9, ADR-0042).
func TestResolveProbeHost(t *testing.T) {
	granted := func(hosts ...string) artefact.TargetInfo {
		set := map[string]bool{}
		for _, host := range hosts {
			set[host] = true
		}
		return artefact.TargetInfo{Hosts: set}
	}
	regions := artefact.ProviderInfo{Enumerations: map[string][]string{
		"region": {"eu-central-1", "us-east-1"},
	}}

	for _, c := range []struct {
		name      string
		provider  artefact.ProviderInfo
		operation artefact.OperationInfo
		target    artefact.TargetInfo
		supplied  string
		want      artefact.ProbeHost
	}{
		{
			name:      "a grant of one host fills it",
			operation: artefact.OperationInfo{HostTemplate: "{from-target}"},
			target:    granted("status.hyper.dev"),
			want:      artefact.ProbeHost{Reach: artefact.ReachGranted, Host: "status.hyper.dev"},
		},
		{
			name:      "a literal host the grant admits",
			operation: artefact.OperationInfo{HostTemplate: "api.cloudflare.com"},
			target:    granted("api.cloudflare.com"),
			want:      artefact.ProbeHost{Reach: artefact.ReachGranted, Host: "api.cloudflare.com"},
		},
		{
			name:      "a literal host the grant does not admit",
			operation: artefact.OperationInfo{HostTemplate: "api.cloudflare.com"},
			target:    granted("status.hyper.dev"),
			want:      artefact.ProbeHost{Reach: artefact.ReachNotGranted, Host: "api.cloudflare.com"},
		},
		{
			name:      "the input the Operation names carries the host",
			operation: artefact.OperationInfo{HostTemplate: "{from-target}", HostInput: "host"},
			target:    granted("status.hyper.dev", "cert.hyper.dev"),
			supplied:  "cert.hyper.dev",
			want:      artefact.ProbeHost{Reach: artefact.ReachGranted, Host: "cert.hyper.dev"},
		},
		{
			name:      "and it is checked for grant membership like any other",
			operation: artefact.OperationInfo{HostTemplate: "{from-target}", HostInput: "host"},
			target:    granted("status.hyper.dev", "cert.hyper.dev"),
			supplied:  "elsewhere.example.com",
			want:      artefact.ProbeHost{Reach: artefact.ReachNotGranted, Host: "elsewhere.example.com"},
		},
		{
			name:      "a grant of nothing at all names no host to decline over",
			operation: artefact.OperationInfo{HostTemplate: "{from-target}"},
			target:    granted(),
			want:      artefact.ProbeHost{Reach: artefact.ReachNotGranted},
		},
		{
			name:      "two granted candidates under no host-input: decide nothing",
			operation: artefact.OperationInfo{HostTemplate: "{from-target}"},
			target:    granted("status.hyper.dev", "cert.hyper.dev"),
			want:      artefact.ProbeHost{Reach: artefact.ReachUndecidable, Granted: 2},
		},
		{
			name:      "an enumeration expands to a candidate set the grant cuts to one",
			provider:  regions,
			operation: artefact.OperationInfo{HostTemplate: "s3.{region}.amazonaws.com"},
			target:    granted("s3.us-east-1.amazonaws.com"),
			want:      artefact.ProbeHost{Reach: artefact.ReachGranted, Host: "s3.us-east-1.amazonaws.com"},
		},
		{
			name:      "an enumeration the grant admits twice decides nothing",
			provider:  regions,
			operation: artefact.OperationInfo{HostTemplate: "s3.{region}.amazonaws.com"},
			target:    granted("s3.us-east-1.amazonaws.com", "s3.eu-central-1.amazonaws.com"),
			want:      artefact.ProbeHost{Reach: artefact.ReachUndecidable, Granted: 2},
		},
		{
			name:      "a hole naming neither from-target nor an enumeration expands to nothing",
			operation: artefact.OperationInfo{HostTemplate: "api.{tenant}.example.com"},
			target:    granted("api.acme.example.com"),
			want:      artefact.ProbeHost{Reach: artefact.ReachIllegible},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := artefact.ResolveProbeHost(c.provider, c.operation, c.target, c.supplied)
			if got != c.want {
				t.Errorf("ResolveProbeHost = %+v, want %+v", got, c.want)
			}
		})
	}
}
