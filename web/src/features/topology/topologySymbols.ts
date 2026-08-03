import bridge from "@/assets/topology/bridge.svg";
import docker from "@/assets/topology/docker.svg";
import nat from "@/assets/topology/nat.svg";
import pc from "@/assets/topology/pc.svg";
import qemu from "@/assets/topology/qemu.svg";
import switchL2 from "@/assets/topology/switch-l2.svg";
import switchL3 from "@/assets/topology/switch-l3.svg";

const symbols: Record<string, string> = {
  qemu,
  docker,
  pc,
  bridge,
  nat_bridge: nat,
  switch_l2: switchL2,
  switch_l3: switchL3,
};

export function topologySymbol(kind: string): string {
  return `image://${symbols[kind] || qemu}`;
}
