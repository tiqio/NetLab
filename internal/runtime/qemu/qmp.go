package qemu

import (
	"context"
	"encoding/json"
	"github.com/digitalocean/go-qemu/qmp"
	"time"
)

type QMP struct{ monitor *qmp.SocketMonitor }

func ConnectQMP(path string, timeout time.Duration) (*QMP, error) {
	monitor, err := qmp.NewSocketMonitor("unix", path, timeout)
	if err != nil {
		return nil, err
	}
	if err = monitor.Connect(); err != nil {
		monitor.Disconnect()
		return nil, err
	}
	return &QMP{monitor: monitor}, nil
}
func (q *QMP) Close() error { return q.monitor.Disconnect() }
func (q *QMP) Run(command string, args any) (json.RawMessage, error) {
	body, err := json.Marshal(qmp.Command{Execute: command, Args: args})
	if err != nil {
		return nil, err
	}
	result, err := q.monitor.Run(body)
	return json.RawMessage(result), err
}
func (q *QMP) Events(ctx context.Context) (<-chan qmp.Event, error) { return q.monitor.Events(ctx) }
