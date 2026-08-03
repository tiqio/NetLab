package qemu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/digitalocean/go-qemu/qmp"
)

type QMPCommander interface {
	Run(string, any) (json.RawMessage, error)
	Events(context.Context) (<-chan qmp.Event, error)
}

type HotplugNIC struct {
	ID         string
	NetdevID   string
	TapName    string
	Driver     string
	MACAddress string
	Bus        string
}

func AddNIC(_ context.Context, monitor QMPCommander, nic HotplugNIC) error {
	if nic.ID == "" || nic.NetdevID == "" || nic.TapName == "" || nic.Driver == "" {
		return fmt.Errorf("hotplug identifiers and driver are required")
	}
	if _, err := monitor.Run("netdev_add", map[string]any{"type": "tap", "id": nic.NetdevID, "ifname": nic.TapName, "script": "no", "downscript": "no", "vnet_hdr": false}); err != nil {
		return err
	}
	arguments := map[string]any{"driver": nic.Driver, "id": nic.ID, "netdev": nic.NetdevID}
	if nic.MACAddress != "" {
		arguments["mac"] = nic.MACAddress
	}
	if nic.Bus != "" {
		arguments["bus"] = nic.Bus
	}
	if _, err := monitor.Run("device_add", arguments); err != nil {
		_, _ = monitor.Run("netdev_del", map[string]any{"id": nic.NetdevID})
		return err
	}
	return nil
}

func RemoveNIC(ctx context.Context, monitor QMPCommander, deviceID, netdevID string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	events, err := monitor.Events(ctx)
	if err != nil {
		return err
	}
	if _, err = monitor.Run("device_del", map[string]any{"id": deviceID}); err != nil {
		return err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	observed := make([]string, 0, 4)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			present, queryErr := devicePresent(monitor, deviceID)
			if queryErr != nil {
				return fmt.Errorf("timed out waiting for DEVICE_DELETED for %s; query PCI state: %w; observed events: %v", deviceID, queryErr, observed)
			}
			if !present {
				_, err = monitor.Run("netdev_del", map[string]any{"id": netdevID})
				return err
			}
			return fmt.Errorf("timed out waiting for DEVICE_DELETED for %s; device is still present; observed events: %v", deviceID, observed)
		case event, ok := <-events:
			if !ok {
				return fmt.Errorf("QMP event stream closed")
			}
			device, _ := event.Data["device"].(string)
			path, _ := event.Data["path"].(string)
			if len(observed) < cap(observed) {
				observed = append(observed, fmt.Sprintf("%s:%v", event.Event, event.Data))
			}
			if event.Event == "DEVICE_DELETED" && (device == deviceID || path == deviceID || strings.HasSuffix(path, "/"+deviceID)) {
				_, err = monitor.Run("netdev_del", map[string]any{"id": netdevID})
				return err
			}
		}
	}
}

func devicePresent(monitor QMPCommander, deviceID string) (bool, error) {
	body, err := monitor.Run("query-pci", nil)
	if err != nil {
		return false, err
	}
	encodedID, _ := json.Marshal(deviceID)
	return strings.Contains(string(body), `"qdev_id":`+string(encodedID)) || strings.Contains(string(body), `"id":`+string(encodedID)), nil
}
