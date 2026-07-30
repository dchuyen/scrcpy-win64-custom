package main

import "testing"

func TestParseDeviceTarget(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantHost  string
		wantPort  int
		wantHave  bool
		wantError bool
	}{
		{name: "bare ip", input: "192.168.1.10", wantHost: "192.168.1.10", wantHave: false},
		{name: "ip with port", input: "192.168.1.10:5555", wantHost: "192.168.1.10", wantPort: 5555, wantHave: true},
		{name: "invalid", input: "not-an-ip", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, havePort, err := parseDeviceTarget(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatalf("parseDeviceTarget(%q) error = nil, want error", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseDeviceTarget(%q) unexpected error: %v", tt.input, err)
			}
			if host != tt.wantHost {
				t.Fatalf("parseDeviceTarget(%q) host = %q, want %q", tt.input, host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Fatalf("parseDeviceTarget(%q) port = %d, want %d", tt.input, port, tt.wantPort)
			}
			if havePort != tt.wantHave {
				t.Fatalf("parseDeviceTarget(%q) havePort = %v, want %v", tt.input, havePort, tt.wantHave)
			}
		})
	}
}
