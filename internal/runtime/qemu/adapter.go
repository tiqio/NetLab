package qemu

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Adapter struct {
	Binary, QEMUImg, Root string
	Acceleration          string
	ReadinessTimeout      time.Duration
	StopTimeout           time.Duration
}
type LaunchManifest struct {
	NodeID       domain.ID `json:"node_id"`
	PID          int       `json:"pid"`
	Acceleration string    `json:"acceleration"`
	Arguments    []string  `json:"arguments"`
	Disk         string    `json:"disk"`
	QMP          string    `json:"qmp"`
	QGA          string    `json:"qga"`
	Serial       string    `json:"serial"`
	VNC          string    `json:"vnc"`
}

func NewAdapter(root string) (*Adapter, error) {
	binary, err := exec.LookPath("qemu-system-x86_64")
	if err != nil {
		return nil, err
	}
	qemuImg, _ := exec.LookPath("qemu-img")
	acceleration := "tcg"
	if info, statErr := os.Stat("/dev/kvm"); statErr == nil && info.Mode()&os.ModeDevice != 0 {
		acceleration = "kvm"
	}
	return &Adapter{Binary: binary, QEMUImg: qemuImg, Root: root, Acceleration: acceleration}, nil
}
func (a *Adapter) RuntimeDir(id domain.ID) string {
	return filepath.Join(a.Root, "runtime", "qemu", string(id))
}
func (a *Adapter) BuildArgs(node domain.Node, imagePath string) ([]string, LaunchManifest) {
	dir := a.RuntimeDir(node.ID)
	acceleration := a.Acceleration
	if acceleration == "" {
		acceleration = "kvm"
	}
	machine := qemuConfigOption(node, "machine", "q35", "pc", "q35")
	cpu := qemuConfigOption(node, "cpu", "host", "host", "max")
	diskInterface := qemuConfigOption(node, "disk_interface", "virtio", "ide", "virtio")
	manifest := LaunchManifest{NodeID: node.ID, Acceleration: acceleration, Disk: imagePath, QMP: filepath.Join(dir, "qmp.sock"), QGA: filepath.Join(dir, "qga.sock"), Serial: filepath.Join(dir, "serial.sock"), VNC: filepath.Join(dir, "vnc.sock")}
	args := []string{"-name", "guest=netlab:" + string(node.ID), "-machine", machine, "-accel", acceleration, "-cpu", cpu}
	if acceleration == "tcg" {
		args = []string{"-name", "guest=netlab:" + string(node.ID), "-machine", machine, "-accel", "tcg,thread=multi", "-cpu", "max"}
	}
	args = append(args, "-smp", strconv.Itoa(max(node.CPUCount, 1)), "-m", strconv.Itoa(max(node.MemoryMiB, 512)), "-nodefaults", "-display", "none", "-qmp", "unix:"+manifest.QMP+",server=on,wait=off", "-chardev", "socket,id=qga,path="+manifest.QGA+",server=on,wait=off", "-device", "virtio-serial-pci", "-device", "virtserialport,chardev=qga,name=org.qemu.guest_agent.0", "-serial", "unix:"+manifest.Serial+",server=on,wait=off")
	if qemuConfigOption(node, "rtc_base", "", "utc") == "utc" {
		args = append(args, "-rtc", "base=utc")
	}
	if hasConsoleMode(node, "vnc") {
		if machine == "q35" {
			args = append(args, "-device", "VGA,bus=pcie.0,addr=2")
		} else {
			args = append(args, "-vga", qemuConfigOption(node, "vga", "std", "std"))
		}
		args = append(args, "-vnc", "unix:"+manifest.VNC)
	}
	args = append(args, "-drive", "file="+imagePath+",if="+diskInterface+",format=qcow2")
	if machine == "q35" {
		for slot := 0; slot < hotplugPortCount(node); slot++ {
			address := 3 + slot/8
			function := slot % 8
			options := fmt.Sprintf("pcie-root-port,id=%s,bus=pcie.0,chassis=%d,slot=%d,port=%d,addr=%x.%x", rootPortID(slot), slot+1, slot+1, slot+1, address, function)
			if function == 0 {
				options += ",multifunction=on"
			}
			args = append(args, "-device", options)
		}
	}
	for _, iface := range linuxnet.InterfaceDescriptors(node) {
		driver := iface.Driver
		if driver == "" {
			driver = "virtio-net-pci"
		}
		netdevID := "net-" + string(iface.ID)
		device := driver + ",netdev=" + netdevID + ",mac=" + iface.MACAddress + ",id=nic-" + string(iface.ID)
		if machine == "q35" {
			device += ",bus=" + rootPortID(iface.Slot)
		}
		args = append(args, "-netdev", "tap,id="+netdevID+",ifname="+linuxnet.HostInterfaceName(iface.ID)+",script=no,downscript=no,vnet_hdr=off", "-device", device)
	}
	if seedPath, ok := node.Config["seed_iso"].(string); ok && seedPath != "" {
		args = append(args, "-smbios", "type=1,serial=ds=nocloud")
		args = append(args, "-drive", "file="+seedPath+",media=cdrom,readonly=on")
	}
	manifest.Arguments = args
	return args, manifest
}

func qemuConfigOption(node domain.Node, key, fallback string, allowed ...string) string {
	value, _ := node.Config[key].(string)
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return fallback
}

func hasConsoleMode(node domain.Node, expected string) bool {
	switch values := node.Config["console_modes"].(type) {
	case []string:
		for _, value := range values {
			if value == expected {
				return true
			}
		}
	case []any:
		for _, raw := range values {
			if value, ok := raw.(string); ok && value == expected {
				return true
			}
		}
	}
	return false
}

func hotplugPortCount(node domain.Node) int {
	count := max(node.InterfaceLimit, len(linuxnet.InterfaceDescriptors(node)))
	if count <= 0 {
		count = 8
	}
	return min(count, 64)
}

func rootPortID(slot int) string {
	return fmt.Sprintf("netlab-rp-%d", slot)
}
func (a *Adapter) Inspect(_ context.Context, node domain.Node) (ports.ActualNode, error) {
	body, err := os.ReadFile(filepath.Join(a.RuntimeDir(node.ID), "launch.json"))
	if os.IsNotExist(err) {
		return ports.ActualNode{State: domain.ObservedStopped}, nil
	}
	if err != nil {
		return ports.ActualNode{}, err
	}
	var manifest LaunchManifest
	if err = json.Unmarshal(body, &manifest); err != nil {
		return ports.ActualNode{}, err
	}
	process, err := os.FindProcess(manifest.PID)
	if err != nil {
		return ports.ActualNode{State: domain.ObservedStopped}, nil
	}
	if err = process.Signal(syscall.Signal(0)); err != nil {
		return ports.ActualNode{State: domain.ObservedStopped}, nil
	}
	return ports.ActualNode{State: domain.ObservedRunning, Owner: map[string]string{"pid": strconv.Itoa(manifest.PID), "qmp": manifest.QMP}}, nil
}
func (a *Adapter) Provision(ctx context.Context, node domain.Node) error {
	baseImagePath, ok := node.Config["image_path"].(string)
	if !ok || baseImagePath == "" {
		return fmt.Errorf("node image_path required")
	}
	dir := a.RuntimeDir(node.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return a.ensureOverlay(ctx, baseImagePath, filepath.Join(dir, "disk.qcow2"), node.StorageGiB)
}
func (a *Adapter) Start(ctx context.Context, node domain.Node) error {
	if err := a.Provision(ctx, node); err != nil {
		return err
	}
	dir := a.RuntimeDir(node.ID)
	imagePath := filepath.Join(dir, "disk.qcow2")
	args, manifest := a.BuildArgs(node, imagePath)
	ipPath, ipErr := exec.LookPath("ip")
	if ipErr != nil {
		return ipErr
	}
	created := make([]string, 0)
	for _, iface := range linuxnet.InterfaceDescriptors(node) {
		name := linuxnet.HostInterfaceName(iface.ID)
		if exec.CommandContext(ctx, ipPath, "link", "show", name).Run() == nil {
			continue
		}
		if output, err := exec.CommandContext(ctx, ipPath, "tuntap", "add", "dev", name, "mode", "tap").CombinedOutput(); err != nil {
			return fmt.Errorf("create TAP %s: %s: %w", name, strings.TrimSpace(string(output)), err)
		}
		created = append(created, name)
		_ = exec.CommandContext(ctx, ipPath, "link", "set", name, "alias", "netlab:"+string(iface.ID)).Run()
		_ = exec.CommandContext(ctx, ipPath, "link", "set", name, "up").Run()
	}
	command := exec.Command(a.Binary, args...)
	command.Env = append(os.Environ(), "NETLAB_OWNERSHIP=node:"+string(node.ID))
	logFile, err := os.OpenFile(filepath.Join(dir, "qemu.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	command.Stdout = logFile
	command.Stderr = logFile
	if err = command.Start(); err != nil {
		logFile.Close()
		for _, name := range created {
			_ = exec.CommandContext(context.Background(), ipPath, "link", "delete", name).Run()
		}
		return err
	}
	manifest.PID = command.Process.Pid
	body, _ := json.MarshalIndent(manifest, "", "  ")
	if err = os.WriteFile(filepath.Join(dir, "launch.json"), body, 0o600); err != nil {
		_ = command.Process.Kill()
		return err
	}
	go func() { _ = command.Wait(); _ = logFile.Close() }()
	if err = a.waitReady(ctx, manifest, command.Process); err != nil {
		_ = command.Process.Kill()
		for _, name := range created {
			_ = exec.Command(ipPath, "link", "delete", name).Run()
		}
		_ = os.Remove(filepath.Join(dir, "launch.json"))
		return err
	}
	return nil
}

func (a *Adapter) waitReady(ctx context.Context, manifest LaunchManifest, process *os.Process) error {
	timeout := a.ReadinessTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var lastError error
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := process.Signal(syscall.Signal(0)); err != nil {
			return fmt.Errorf("QEMU exited before QMP became ready: %s", a.recentLog(manifest.NodeID))
		}
		monitor, err := ConnectQMP(manifest.QMP, 250*time.Millisecond)
		if err == nil {
			_ = monitor.Close()
			return nil
		}
		lastError = err
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("QMP readiness timed out: %v: %s", lastError, a.recentLog(manifest.NodeID))
}

func (a *Adapter) recentLog(id domain.ID) string {
	body, err := os.ReadFile(filepath.Join(a.RuntimeDir(id), "qemu.log"))
	if err != nil {
		return "QEMU log unavailable"
	}
	const maximum = 8192
	if len(body) > maximum {
		body = body[len(body)-maximum:]
	}
	message := strings.TrimSpace(string(body))
	if message == "" {
		return "QEMU produced no diagnostic output"
	}
	return message
}

func (a *Adapter) ensureOverlay(ctx context.Context, base, destination string, sizeGiB int) error {
	if _, err := os.Stat(destination); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	temporary := destination + ".tmp"
	_ = os.Remove(temporary)
	if err := a.CreateOverlay(ctx, base, temporary, sizeGiB); err != nil {
		return err
	}
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}
func (a *Adapter) Stop(ctx context.Context, node domain.Node) error {
	actual, err := a.Inspect(ctx, node)
	if err != nil || actual.State != domain.ObservedRunning {
		return err
	}
	pid, _ := strconv.Atoi(actual.Owner["pid"])
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err = process.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	timeout := a.StopTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if process.Signal(syscall.Signal(0)) != nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return process.Kill()
}
func (a *Adapter) Delete(ctx context.Context, node domain.Node) error {
	if err := a.Stop(ctx, node); err != nil {
		return err
	}
	if ipPath, err := exec.LookPath("ip"); err == nil {
		for _, iface := range linuxnet.InterfaceDescriptors(node) {
			_ = exec.CommandContext(ctx, ipPath, "link", "delete", linuxnet.HostInterfaceName(iface.ID)).Run()
		}
	}
	return os.RemoveAll(a.RuntimeDir(node.ID))
}
func (a *Adapter) CreateOverlay(ctx context.Context, base, destination string, sizeGiB int) error {
	if a.QEMUImg == "" {
		return fmt.Errorf("qemu-img unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	command := exec.CommandContext(ctx, a.QEMUImg, "create", "-f", "qcow2", "-F", "qcow2", "-b", base, destination)
	if sizeGiB > 0 {
		command.Args = append(command.Args, fmt.Sprintf("%dG", sizeGiB))
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-img create: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (a *Adapter) HotAddInterface(ctx context.Context, node domain.Node, iface domain.Interface, tapName string) error {
	monitor, err := ConnectQMP(filepath.Join(a.RuntimeDir(node.ID), "qmp.sock"), 5*time.Second)
	if err != nil {
		return err
	}
	defer monitor.Close()
	return AddNIC(ctx, monitor, HotplugNIC{ID: "nic-" + string(iface.ID), NetdevID: "net-" + string(iface.ID), TapName: tapName, Driver: iface.Driver, MACAddress: iface.MACAddress, Bus: hotplugBus(node, iface.Slot)})
}

func (a *Adapter) InterfaceTapName(iface domain.Interface) string {
	return linuxnet.HostInterfaceName(iface.ID)
}

func hotplugBus(node domain.Node, slot int) string {
	if qemuConfigOption(node, "machine", "q35", "pc", "q35") == "pc" {
		return ""
	}
	return rootPortID(slot)
}

func (a *Adapter) HotRemoveInterface(ctx context.Context, node domain.Node, iface domain.Interface) error {
	monitor, err := ConnectQMP(filepath.Join(a.RuntimeDir(node.ID), "qmp.sock"), 5*time.Second)
	if err != nil {
		return err
	}
	defer monitor.Close()
	return RemoveNIC(ctx, monitor, "nic-"+string(iface.ID), "net-"+string(iface.ID), 15*time.Second)
}

func (a *Adapter) GuestExec(ctx context.Context, node domain.Node, request GuestExecRequest) (GuestExecResult, error) {
	agent, err := ConnectGuestAgent(filepath.Join(a.RuntimeDir(node.ID), "qga.sock"), 5*time.Second)
	if err != nil {
		return GuestExecResult{}, err
	}
	defer agent.Close()
	return ExecuteGuest(ctx, agent, request)
}

var _ ports.NodeRuntime = (*Adapter)(nil)
