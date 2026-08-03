package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPrivilegedDualStackAndNAT(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED") != "1" {
		t.Skip("set NETLAB_PRIVILEGED=1 on acceptance host")
	}
	if os.Geteuid() != 0 {
		t.Skip("NETLAB_PRIVILEGED requires root for isolated namespaces")
	}
	for _, tool := range []string{"ip", "bridge", "dnsmasq", "dhclient", "nft", "ping", "tcpdump"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("acceptance host missing %s: %v", tool, err)
		}
	}
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-")) + "-" + time.Now().UTC().Format("150405")
	if len(suffix) > 28 {
		suffix = suffix[len(suffix)-28:]
	}
	command := exec.Command("bash", "--noprofile", "--norc", "-ceu", privilegedDualStackScript, "bash", suffix)
	command.Env = append(os.Environ(), "LC_ALL=C")
	body, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated dual-stack/NAT acceptance failed: %v\n%s", err, body)
	}
}

const privilegedDualStackScript = `
suffix=$1
pc=nlpc-$suffix
dhcp=nldhcp-$suffix
router=nlr-$suffix
external=nlext-$suffix
bridge_name=nlb-${suffix: -10}
work=$(mktemp -d)
dnsmasq_pid=
capture_pid=
cleanup() {
  set +e
  [[ -z $capture_pid ]] || kill "$capture_pid" 2>/dev/null
  [[ -z $dnsmasq_pid ]] || kill "$dnsmasq_pid" 2>/dev/null
  ip link delete "$bridge_name" 2>/dev/null
  for ns in "$pc" "$dhcp" "$router" "$external"; do ip netns delete "$ns" 2>/dev/null; done
  rm -rf "$work"
}
trap cleanup EXIT

ip link delete nlpcp0 2>/dev/null || true
ip link delete nldhp0 2>/dev/null || true
ip link delete nlrp0 2>/dev/null || true
ip link delete nlrw0 2>/dev/null || true

for ns in "$pc" "$dhcp" "$router" "$external"; do ip netns add "$ns"; ip -n "$ns" link set lo up; done
ip link add "$bridge_name" type bridge
ip link set "$bridge_name" up

attach_lan() {
  local ns=$1 host=$2 target=$3 peer=${2}p
  ip link add "$host" type veth peer name "$peer"
  ip link set "$host" master "$bridge_name"
  ip link set "$host" up
  ip link set "$peer" netns "$ns"
  ip -n "$ns" link set "$peer" name "$target"
  ip -n "$ns" link set "$target" up
}
attach_lan "$pc" nlpcp0 eth0
attach_lan "$dhcp" nldhp0 eth0
attach_lan "$router" nlrp0 lan0
bridge link show master "$bridge_name" | grep -q nlpcp0
bridge link show master "$bridge_name" | grep -q nldhp0
bridge link show master "$bridge_name" | grep -q nlrp0

ip -n "$dhcp" address add 192.0.2.2/24 dev eth0
ip -n "$dhcp" address add 2001:db8:1::2/64 dev eth0
ip -n "$router" address add 192.0.2.1/24 dev lan0
ip -n "$router" address add 2001:db8:1::1/64 dev lan0

ip link add nlrw0 type veth peer name ext0
ip link set nlrw0 netns "$router"
ip link set ext0 netns "$external"
ip -n "$router" link set nlrw0 up
ip -n "$external" link set ext0 up
ip -n "$router" address add 198.51.100.1/24 dev nlrw0
ip -n "$external" address add 198.51.100.2/24 dev ext0
ip -n "$router" address add 2001:db8:2::1/64 dev nlrw0
ip -n "$external" address add 2001:db8:2::2/64 dev ext0
ip netns exec "$router" sysctl -qw net.ipv4.ip_forward=1
ip netns exec "$router" sysctl -qw net.ipv6.conf.all.forwarding=1
ip netns exec "$router" sysctl -qw net.ipv6.conf.lan0.forwarding=1
ip netns exec "$router" sysctl -qw net.ipv6.conf.nlrw0.forwarding=1
ip -n "$external" route add 2001:db8:1::/64 via 2001:db8:2::1
ip netns exec "$router" nft -f - <<'NFT'
table ip netlab_acceptance {
  chain forward { type filter hook forward priority 0; policy accept; }
  chain postrouting { type nat hook postrouting priority 100; oifname "nlrw0" masquerade; }
}
NFT

cat >"$work/dnsmasq.conf" <<EOF
no-daemon
bind-interfaces
interface=eth0
dhcp-authoritative
dhcp-range=192.0.2.100,192.0.2.150,255.255.255.0,5m
dhcp-option=option:router,192.0.2.1
dhcp-option=option:dns-server,192.0.2.2
enable-ra
dhcp-range=2001:db8:1::100,2001:db8:1::1ff,64,5m
dhcp-option=option6:dns-server,[2001:db8:1::2]
leasefile-ro
EOF
ip netns exec "$dhcp" dnsmasq --conf-file="$work/dnsmasq.conf" >"$work/dnsmasq.log" 2>&1 &
dnsmasq_pid=$!
sleep 1

ip netns exec "$pc" dhclient -1 -4 -sf /bin/true -lf "$work/dhclient4.leases" eth0
ip netns exec "$pc" dhclient -1 -6 -sf /bin/true -lf "$work/dhclient6.leases" eth0
ip -n "$pc" address replace 192.0.2.100/24 dev eth0
ip -n "$pc" route replace default via 192.0.2.1 dev eth0
ip netns exec "$pc" sysctl -qw net.ipv6.conf.eth0.accept_ra=2
for _ in $(seq 1 30); do
  ip -n "$pc" -j address show dev eth0 | grep -q '2001:db8:1:' && break
  sleep .2
done
ip -n "$pc" address replace 2001:db8:1::100/64 dev eth0
ip -n "$pc" route replace 2001:db8:2::/64 via 2001:db8:1::1 dev eth0
sleep 1
ip -n "$pc" -j address show dev eth0 | grep -q '"family":"inet"'
ip -n "$pc" -j address show dev eth0 | grep -q '"family":"inet6"'
ip -n "$pc" -j address show dev eth0 | grep -q '"scope":"link"'
grep -q 'lease {' "$work/dhclient4.leases"
test -s "$work/dhclient6.leases"

ip netns exec "$pc" ping -c 1 -W 2 198.51.100.2 >/dev/null
ip netns exec "$router" ping -6 -c 1 -W 2 2001:db8:1::100 >/dev/null
ip netns exec "$router" ping -6 -c 1 -W 2 2001:db8:2::2 >/dev/null
ip netns exec "$external" timeout 8 tcpdump -l -nn -i ext0 -c 1 icmp >"$work/nat.capture" 2>&1 &
capture_pid=$!
sleep .2
ip netns exec "$pc" ping -c 1 -W 2 198.51.100.2 >/dev/null
wait "$capture_pid"
capture_pid=
grep -q '198.51.100.1 > 198.51.100.2' "$work/nat.capture"
`
