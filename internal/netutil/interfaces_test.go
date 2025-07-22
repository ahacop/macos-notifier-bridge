package netutil

import (
	"net"
	"testing"
)

func TestIsVMBridge(t *testing.T) {
	tests := []struct {
		name      string
		ifaceName string
		want      bool
	}{
		{"virbr0", "virbr0", true},
		{"virbr1", "virbr1", true},
		{"vmnet1", "vmnet1", true},
		{"vmnet8", "vmnet8", true},
		{"vboxnet0", "vboxnet0", true},
		{"docker0", "docker0", true},
		{"br-custom", "br-abcd1234", true},
		{"uppercase VIRBR0", "VIRBR0", true},
		{"eth0", "eth0", false},
		{"lo", "lo", false},
		{"wlan0", "wlan0", false},
		{"enp0s3", "enp0s3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVMBridge(tt.ifaceName); got != tt.want {
				t.Errorf("isVMBridge(%q) = %v, want %v", tt.ifaceName, got, tt.want)
			}
		})
	}
}

func TestIsVMSubnet(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"libvirt default", "192.168.122.50", true},
		{"vbox host-only", "192.168.56.10", true},
		{"docker default", "172.17.0.2", true},
		{"vbox NAT", "10.0.2.15", true},
		{"private VM range", "172.18.0.5", true},
		{"localhost", "127.0.0.1", false},
		{"public IP", "8.8.8.8", false},
		{"local network", "192.168.1.100", false},
		{"invalid IP", "not-an-ip", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVMSubnet(tt.ip); got != tt.want {
				t.Errorf("IsVMSubnet(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestGetAllBindAddresses(t *testing.T) {
	t.Run("without VM bridges", func(t *testing.T) {
		addrs, err := GetAllBindAddresses(false)
		if err != nil {
			t.Fatalf("GetAllBindAddresses(false) error = %v", err)
		}
		if len(addrs) != 1 || addrs[0] != "localhost" {
			t.Errorf("GetAllBindAddresses(false) = %v, want [localhost]", addrs)
		}
	})

	t.Run("with VM bridges", func(t *testing.T) {
		addrs, err := GetAllBindAddresses(true)
		if err != nil {
			t.Fatalf("GetAllBindAddresses(true) error = %v", err)
		}
		// Should at least have localhost
		if len(addrs) < 1 || addrs[0] != "localhost" {
			t.Errorf("GetAllBindAddresses(true) should start with localhost, got %v", addrs)
		}
		// Note: actual VM bridge detection depends on system configuration
		t.Logf("Detected addresses: %v", addrs)
	})
}

func TestDetectVMBridges(t *testing.T) {
	// This test is system-dependent
	bridges, err := DetectVMBridges()
	if err != nil {
		t.Fatalf("DetectVMBridges() error = %v", err)
	}

	// Log what was found for debugging
	t.Logf("Detected VM bridges: %v", bridges)

	// Verify returned addresses are valid IPs
	for _, addr := range bridges {
		if net.ParseIP(addr) == nil {
			t.Errorf("Invalid IP address returned: %q", addr)
		}
	}
}
