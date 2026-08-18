package firewall

import (
	"fmt"
	"strings"
	"testing"

	"firecrackmanager/internal/database"
)

func TestApplyPortForwardAllProtocols(t *testing.T) {
	mgr := NewManager(t.Logf)
	var commands []string
	mgr.run = func(name string, args ...string) ([]byte, error) {
		cmd := name + " " + strings.Join(args, " ")
		commands = append(commands, cmd)
		if strings.Contains(cmd, " -L ") {
			return []byte(""), nil
		}
		return []byte(""), nil
	}

	netObj := &database.Network{
		ID:           "1234567890",
		BridgeName:   "fcbr123456",
		Subnet:       "192.168.100.0/24",
		OutInterface: "eth0",
	}
	rules := []*database.FirewallRule{{
		ID:        "abcdef1234",
		NetworkID: netObj.ID,
		RuleType:  RuleTypePortForward,
		DestIP:    "192.168.100.10",
		HostPort:  8080,
		DestPort:  80,
		Protocol:  "all",
		Enabled:   true,
	}}

	if err := mgr.ApplyNetworkRules(netObj, rules); err != nil {
		t.Fatalf("ApplyNetworkRules() error = %v", err)
	}

	if !hasCommand(commands, "-p tcp --dport 8080") {
		t.Fatalf("missing tcp DNAT command, commands: %v", commands)
	}
	if !hasCommand(commands, "-p udp --dport 8080") {
		t.Fatalf("missing udp DNAT command, commands: %v", commands)
	}
}

func TestApplyNetworkRulesReturnsCommandFailure(t *testing.T) {
	mgr := NewManager(t.Logf)
	mgr.run = func(name string, args ...string) ([]byte, error) {
		cmd := name + " " + strings.Join(args, " ")
		if strings.Contains(cmd, " -L ") {
			return []byte(""), nil
		}
		if strings.Contains(cmd, "--to-destination") {
			return []byte(""), fmt.Errorf("iptables failed")
		}
		return []byte(""), nil
	}

	netObj := &database.Network{
		ID:           "1234567890",
		BridgeName:   "fcbr123456",
		Subnet:       "192.168.100.0/24",
		OutInterface: "eth0",
	}
	rules := []*database.FirewallRule{{
		ID:        "abcdef1234",
		NetworkID: netObj.ID,
		RuleType:  RuleTypePortForward,
		DestIP:    "192.168.100.10",
		HostPort:  8080,
		DestPort:  80,
		Protocol:  "tcp",
		Enabled:   true,
	}}

	if err := mgr.ApplyNetworkRules(netObj, rules); err == nil {
		t.Fatal("ApplyNetworkRules() error = nil, want command failure")
	}
}

func hasCommand(commands []string, part string) bool {
	for _, cmd := range commands {
		if strings.Contains(cmd, part) {
			return true
		}
	}
	return false
}
