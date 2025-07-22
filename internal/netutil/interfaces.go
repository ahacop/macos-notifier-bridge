// Package netutil provides network utility functions for detecting VM bridge interfaces.
package netutil

import (
	"fmt"
	"net"
	"strings"
)

// VMBridgePatterns contains patterns to identify VM bridge interfaces
var VMBridgePatterns = []string{
	"virbr",   // libvirt/KVM/QEMU
	"vmnet",   // VMware
	"vboxnet", // VirtualBox
	"docker0", // Docker default bridge
	"br-",     // Docker custom bridges
}

// DetectVMBridges returns a list of IP addresses from VM bridge interfaces
func DetectVMBridges() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	var addresses []string
	for _, iface := range interfaces {
		// Skip interfaces that are down
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Check if this is a VM bridge interface
		if !isVMBridge(iface.Name) {
			continue
		}

		// Get addresses for this interface
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			// Extract IP from address
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			// Only include IPv4 addresses
			if ip.To4() == nil {
				continue
			}

			// Skip loopback addresses on bridge interfaces
			if ip.IsLoopback() {
				continue
			}

			addresses = append(addresses, ip.String())
		}
	}

	return addresses, nil
}

// isVMBridge checks if an interface name matches VM bridge patterns
func isVMBridge(name string) bool {
	lowerName := strings.ToLower(name)
	for _, pattern := range VMBridgePatterns {
		if strings.HasPrefix(lowerName, pattern) {
			return true
		}
	}
	return false
}

// GetAllBindAddresses returns localhost plus all detected VM bridge addresses
func GetAllBindAddresses(includeVMBridges bool) ([]string, error) {
	// Always include localhost
	addresses := []string{"localhost"}

	if includeVMBridges {
		bridgeAddrs, err := DetectVMBridges()
		if err != nil {
			return addresses, fmt.Errorf("failed to detect VM bridges: %w", err)
		}
		addresses = append(addresses, bridgeAddrs...)
	}

	return addresses, nil
}

// IsVMSubnet checks if an IP belongs to common VM subnets
func IsVMSubnet(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Common VM subnet ranges
	vmSubnets := []string{
		"192.168.122.0/24", // libvirt default
		"192.168.56.0/24",  // VirtualBox host-only
		"172.17.0.0/16",    // Docker default
		"172.16.0.0/12",    // General private range often used by VMs
		"10.0.2.0/24",      // VirtualBox NAT
	}

	for _, subnet := range vmSubnets {
		_, ipNet, err := net.ParseCIDR(subnet)
		if err != nil {
			continue
		}
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	return false
}
