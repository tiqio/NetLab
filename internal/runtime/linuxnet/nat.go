package linuxnet

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"os/exec"
	"strings"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type NATRuntime struct {
	executor     CommandExecutor
	ip, nft      string
	dhcp         *DNSMasqManager
	observations map[domain.ID]domain.NATServiceObservation
}

func (r *NATRuntime) SetDHCPManager(manager *DNSMasqManager) { r.dhcp = manager }
func (r *NATRuntime) HelperObservation(id domain.ID) (domain.NATServiceObservation, bool) {
	value, ok := r.observations[id]
	return value, ok
}

func (r *NATRuntime) Diagnostics(ctx context.Context, id domain.ID) (map[string]any, error) {
	bridge := ownership.Name("nlnat", id, 15)
	addresses, err := r.executor.Output(ctx, r.ip, "-j", "address", "show", "dev", bridge)
	if err != nil {
		return nil, err
	}
	rules, err := r.executor.Output(ctx, r.nft, "-j", "list", "table", "inet", "netlab_nat")
	if err != nil {
		return nil, err
	}
	forwardRules, forwardErr := r.executor.Output(ctx, r.nft, "-j", "list", "chain", "ip", "filter", "FORWARD")
	link, linkErr := r.executor.Output(ctx, r.ip, "-j", "link", "show", "dev", bridge)
	forwardBody := strings.TrimSpace(string(forwardRules))
	inboundRule := forwardErr == nil && strings.Contains(forwardBody, "netlab-forward-in:"+string(id))
	forwardStatus := map[string]any{
		"outbound_rule":  forwardErr == nil && strings.Contains(forwardBody, "netlab-forward-out:"+string(id)),
		"inbound_rule":   inboundRule,
		"inbound_policy": "all_states",
		"return_rule":    inboundRule,
	}
	if forwardErr == nil {
		forwardStatus["rules"] = json.RawMessage(forwardBody)
	} else {
		forwardStatus["error"] = forwardErr.Error()
	}
	diagnostics := map[string]any{"bridge": bridge, "addresses": json.RawMessage(strings.TrimSpace(string(addresses))), "allocation_status": "gateway_assigned", "external_attachment": map[string]any{"status": statusFor(linkErr), "link": json.RawMessage(strings.TrimSpace(string(link)))}, "nat_rules": json.RawMessage(strings.TrimSpace(string(rules))), "translation_status": map[string]any{"masquerade": strings.Contains(string(rules), "masquerade"), "owned_rule": strings.Contains(string(rules), "netlab:"+string(id))}, "forwarding_status": forwardStatus, "cleanup_status": "owned"}
	if observation, ok := r.HelperObservation(id); ok {
		diagnostics["helper"] = observation
	} else {
		diagnostics["helper"] = map[string]any{"state": "not_observed"}
	}
	return diagnostics, nil
}

func statusFor(err error) string {
	if err == nil {
		return "attached"
	}
	return "missing"
}

type BridgeRuntime struct {
	executor CommandExecutor
	ip       string
	bridge   string
}

func NewBridgeRuntime(executor CommandExecutor) (*BridgeRuntime, error) {
	if executor != nil {
		return &BridgeRuntime{executor: executor, ip: "ip", bridge: "bridge"}, nil
	}
	ip, err := exec.LookPath("ip")
	if err != nil {
		return nil, err
	}
	bridge, err := exec.LookPath("bridge")
	if err != nil {
		return nil, err
	}
	return &BridgeRuntime{executor: SystemExecutor{}, ip: ip, bridge: bridge}, nil
}

func (r *BridgeRuntime) Configure(ctx context.Context, object domain.NetworkObject) error {
	var config domain.BridgeConfig
	if err := decodeConfig(object.Config, &config); err != nil {
		return err
	}
	name := ownership.Name("nlbr", object.ID, 15)
	_ = r.executor.Run(ctx, r.ip, "link", "add", name, "type", "bridge")
	_ = r.executor.Run(ctx, r.ip, "link", "set", "dev", name, "alias", "netlab:"+string(object.ID))
	if config.MTU > 0 {
		if config.MTU < 576 || config.MTU > 9216 {
			return fmt.Errorf("bridge MTU out of range")
		}
		if err := r.executor.Run(ctx, r.ip, "link", "set", "dev", name, "mtu", fmt.Sprint(config.MTU)); err != nil {
			return err
		}
	}
	stpState := "0"
	if config.STP {
		stpState = "1"
	}
	if err := r.executor.Run(ctx, r.ip, "link", "set", "dev", name, "type", "bridge", "stp_state", stpState); err != nil {
		return fmt.Errorf("configure bridge STP: %w", err)
	}
	return r.executor.Run(ctx, r.ip, "link", "set", name, "up")
}

func (r *BridgeRuntime) Diagnostics(ctx context.Context, id domain.ID) (map[string]any, error) {
	name := ownership.Name("nlbr", id, 15)
	link, err := r.executor.Output(ctx, r.ip, "-d", "-j", "link", "show", "dev", name)
	if err != nil {
		return nil, fmt.Errorf("inspect bridge link: %w", err)
	}
	ports, err := r.executor.Output(ctx, r.ip, "-d", "-j", "link", "show", "master", name)
	if err != nil {
		return nil, fmt.Errorf("inspect bridge ports: %w", err)
	}
	fdb, err := r.executor.Output(ctx, r.bridge, "-j", "fdb", "show", "br", name)
	if err != nil {
		return nil, fmt.Errorf("inspect bridge forwarding database: %w", err)
	}
	linkBody := strings.TrimSpace(string(link))
	return map[string]any{
		"bridge":         name,
		"link":           json.RawMessage(linkBody),
		"ports":          json.RawMessage(strings.TrimSpace(string(ports))),
		"forwarding_db":  json.RawMessage(strings.TrimSpace(string(fdb))),
		"stp_enabled":    strings.Contains(linkBody, `"stp_state":1`),
		"cleanup_status": "owned",
	}, nil
}

func (r *BridgeRuntime) Delete(ctx context.Context, id domain.ID) error {
	return r.executor.Run(ctx, r.ip, "link", "delete", ownership.Name("nlbr", id, 15))
}

func NewNATRuntime(executor CommandExecutor) (*NATRuntime, error) {
	if executor != nil {
		return &NATRuntime{executor: executor, ip: "ip", nft: "nft", observations: map[domain.ID]domain.NATServiceObservation{}}, nil
	}
	ip, err := exec.LookPath("ip")
	if err != nil {
		return nil, err
	}
	nft, err := exec.LookPath("nft")
	if err != nil {
		return nil, err
	}
	return &NATRuntime{executor: SystemExecutor{}, ip: ip, nft: nft, observations: map[domain.ID]domain.NATServiceObservation{}}, nil
}
func (r *NATRuntime) Configure(ctx context.Context, object domain.NetworkObject) error {
	var config domain.NATConfig
	if err := decodeConfig(object.Config, &config); err != nil {
		return err
	}
	if err := domain.ValidateNATConfig(config); err != nil {
		return err
	}
	uplink, err := r.resolveUplink(ctx, config.Uplink)
	if err != nil {
		return err
	}
	bridge := ownership.Name("nlnat", object.ID, 15)
	if err := r.executor.Run(ctx, r.ip, "link", "add", bridge, "type", "bridge"); err != nil {
		if _, inspectErr := r.executor.Output(ctx, r.ip, "link", "show", bridge); inspectErr != nil {
			return err
		}
	}
	_ = r.executor.Run(ctx, r.ip, "link", "set", "dev", bridge, "alias", "netlab:"+string(object.ID))
	prefix, _ := netip.ParsePrefix(config.IPv4Prefix)
	gateway := prefix.Addr().Next()
	if err := r.executor.Run(ctx, r.ip, "address", "replace", fmt.Sprintf("%s/%d", gateway, prefix.Bits()), "dev", bridge); err != nil {
		return err
	}
	if err := r.executor.Run(ctx, r.ip, "link", "set", bridge, "up"); err != nil {
		return err
	}
	_ = r.executor.Run(ctx, r.nft, "add", "table", "inet", "netlab_nat")
	_ = r.executor.Run(ctx, r.nft, "add", "chain", "inet", "netlab_nat", "postrouting", "{", "type", "nat", "hook", "postrouting", "priority", "srcnat", ";", "}")
	if body, inspectErr := r.executor.Output(ctx, r.nft, "-a", "list", "chain", "inet", "netlab_nat", "postrouting"); inspectErr == nil {
		r.deleteOwnedRules(ctx, body, "netlab:"+string(object.ID), "inet", "netlab_nat", "postrouting")
	}
	if body, inspectErr := r.executor.Output(ctx, r.nft, "-a", "list", "chain", "ip", "filter", "FORWARD"); inspectErr == nil {
		r.deleteOwnedRules(ctx, body, "netlab-forward-out:"+string(object.ID), "ip", "filter", "FORWARD")
		r.deleteOwnedRules(ctx, body, "netlab-forward-in:"+string(object.ID), "ip", "filter", "FORWARD")
	}
	if err := r.executor.Run(ctx, r.nft, "add", "rule", "inet", "netlab_nat", "postrouting", "ip", "saddr", config.IPv4Prefix, "oifname", uplink, "masquerade", "comment", `"netlab:`+string(object.ID)+`"`); err != nil {
		return err
	}
	if err := r.ensureForwardRules(ctx, object.ID, bridge, uplink); err != nil {
		return err
	}
	return r.configureDHCP(ctx, object, config)
}

func (r *NATRuntime) resolveUplink(ctx context.Context, configured string) (string, error) {
	if configured != "auto" {
		return configured, nil
	}
	body, err := r.executor.Output(ctx, r.ip, "route", "show", "default")
	if err != nil {
		return "", fmt.Errorf("resolve NAT uplink from default route: %w", err)
	}
	fields := strings.Fields(string(body))
	for index := range fields {
		if fields[index] == "dev" && index+1 < len(fields) {
			return fields[index+1], nil
		}
	}
	return "", fmt.Errorf("resolve NAT uplink: default route has no device")
}

func (r *NATRuntime) ensureForwardRules(ctx context.Context, id domain.ID, bridge, uplink string) error {
	body, err := r.executor.Output(ctx, r.nft, "list", "chain", "ip", "filter", "FORWARD")
	if err != nil {
		return fmt.Errorf("inspect host FORWARD chain: %w", err)
	}
	outComment := "netlab-forward-out:" + string(id)
	inComment := "netlab-forward-in:" + string(id)
	if !strings.Contains(string(body), outComment) {
		if err := r.executor.Run(ctx, r.nft, "insert", "rule", "ip", "filter", "FORWARD", "iifname", bridge, "oifname", uplink, "accept", "comment", `"`+outComment+`"`); err != nil {
			return fmt.Errorf("allow NAT outbound forwarding: %w", err)
		}
	}
	if !strings.Contains(string(body), inComment) {
		if err := r.executor.Run(ctx, r.nft, "insert", "rule", "ip", "filter", "FORWARD", "iifname", uplink, "oifname", bridge, "accept", "comment", `"`+inComment+`"`); err != nil {
			return fmt.Errorf("allow NAT inbound forwarding: %w", err)
		}
	}
	return nil
}
func (r *NATRuntime) configureDHCP(ctx context.Context, object domain.NetworkObject, config domain.NATConfig) error {
	if r.dhcp == nil {
		if config.DHCPv4 != nil || config.DHCPv6 != nil || config.RouterAdvertisements {
			return fmt.Errorf("dnsmasq helper unavailable")
		}
		return nil
	}
	observation, err := r.dhcp.Start(ctx, object, config)
	if err == nil {
		r.observations[object.ID] = observation
	}
	return err
}
func (r *NATRuntime) Delete(ctx context.Context, id domain.ID) error {
	if r.dhcp != nil {
		_ = r.dhcp.Stop(ctx, id)
	}
	delete(r.observations, id)
	body, err := r.executor.Output(ctx, r.nft, "-a", "list", "chain", "inet", "netlab_nat", "postrouting")
	if err == nil {
		r.deleteOwnedRules(ctx, body, "netlab:"+string(id), "inet", "netlab_nat", "postrouting")
	}
	if body, err := r.executor.Output(ctx, r.nft, "-a", "list", "chain", "ip", "filter", "FORWARD"); err == nil {
		r.deleteOwnedRules(ctx, body, "netlab-forward-out:"+string(id), "ip", "filter", "FORWARD")
		r.deleteOwnedRules(ctx, body, "netlab-forward-in:"+string(id), "ip", "filter", "FORWARD")
	}
	bridge := ownership.Name("nlnat", id, 15)
	_ = r.executor.Run(ctx, r.ip, "link", "delete", bridge)
	return nil
}

func (r *NATRuntime) deleteOwnedRules(ctx context.Context, body []byte, comment, family, table, chain string) {
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, comment) {
			continue
		}
		fields := strings.Fields(line)
		for index := range fields {
			if fields[index] == "handle" && index+1 < len(fields) {
				_ = r.executor.Run(ctx, r.nft, "delete", "rule", family, table, chain, "handle", fields[index+1])
				break
			}
		}
	}
}
