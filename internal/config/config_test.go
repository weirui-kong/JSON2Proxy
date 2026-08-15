package config

import (
	"strings"
	"testing"
)

func TestServerDefaultsMaxConnections(t *testing.T) {
	cfg := Server{Users: []User{{ClientID: "user"}}}
	cfg.Defaults()
	if got := cfg.Users[0].MaxConnections; got != 4096 {
		t.Fatalf("default max_connections = %d, want 4096", got)
	}
}

func TestPerformanceModeDefaultsAndValidation(t *testing.T) {
	cfg := Server{}
	cfg.Defaults()
	if cfg.Performance.Mode != "interactive" {
		t.Fatalf("default performance mode = %q", cfg.Performance.Mode)
	}
	if err := (Performance{Mode: "throughput"}).Validate(); err != nil {
		t.Fatalf("throughput mode rejected: %v", err)
	}
	if err := (Performance{Mode: "invalid"}).Validate(); err == nil {
		t.Fatal("invalid performance mode accepted")
	}
}

func TestServerDefaultsOutboundDNS(t *testing.T) {
	cfg := Server{Outbound: Outbound{Type: "socks5", Address: "127.0.0.1:1080"}}
	cfg.Defaults()
	if cfg.Outbound.DNS != "local" {
		t.Fatalf("default outbound DNS = %q, want local", cfg.Outbound.DNS)
	}
}

func TestValidateOutbound(t *testing.T) {
	valid := Outbound{
		Type: "socks5", Address: "[2001:db8::1]:1080",
		Username: "user", Password: "pass", DNS: "local",
	}
	if err := validateOutbound(valid); err != nil {
		t.Fatalf("valid outbound rejected: %v", err)
	}
	for name, value := range map[string]Outbound{
		"unsupported type": {Type: "http", Address: "127.0.0.1:8080", DNS: "local"},
		"missing port":     {Type: "socks5", Address: "127.0.0.1", DNS: "local"},
		"remote DNS":       {Type: "socks5", Address: "127.0.0.1:1080", DNS: "remote"},
		"password only":    {Type: "socks5", Address: "127.0.0.1:1080", Password: "secret", DNS: "local"},
		"username only":    {Type: "socks5", Address: "127.0.0.1:1080", Username: "user", DNS: "local"},
		"oversize user":    {Type: "socks5", Address: "127.0.0.1:1080", Username: strings.Repeat("x", 256), DNS: "local"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateOutbound(value); err == nil {
				t.Fatal("invalid outbound accepted")
			}
		})
	}
}

func TestServerDefaultsPreservesExplicitMaxConnections(t *testing.T) {
	cfg := Server{Users: []User{{ClientID: "user", MaxConnections: 128}}}
	cfg.Defaults()
	if got := cfg.Users[0].MaxConnections; got != 128 {
		t.Fatalf("explicit max_connections changed to %d", got)
	}
}

func TestUDPDefaultsEnabledAndFalseIsPreserved(t *testing.T) {
	server := Server{}
	server.Defaults()
	if !server.UDP.IsEnabled() {
		t.Fatal("server UDP must default to enabled")
	}
	client := Client{SocksListen: "127.0.0.1:8444"}
	client.Defaults()
	if !client.UDP.IsEnabled() || client.UDP.Bind != "127.0.0.1:0" {
		t.Fatalf("unexpected client UDP defaults: enabled=%t bind=%q", client.UDP.IsEnabled(), client.UDP.Bind)
	}
	disabled := false
	server = Server{UDP: ServerUDP{Enabled: &disabled}}
	server.Defaults()
	client = Client{SocksListen: "127.0.0.1:8444", UDP: ClientUDP{Enabled: &disabled}}
	client.Defaults()
	if server.UDP.IsEnabled() || client.UDP.IsEnabled() {
		t.Fatal("explicit UDP disable was overwritten by defaults")
	}
}
