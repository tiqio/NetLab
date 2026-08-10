import { describe, expect, it, vi } from "vitest";
import type { NetworkObjectLink, NetworkObjectLinkTaskEnvelope } from "@/api";
import { runObjectLinkDeletion } from "./objectLinkDeletion";
import {
  captureTopologyCreateWorkspace,
  openTopologyCreateDrawer,
} from "./topologyCreateDrawerState";
import { applyAuthoritativeCreation } from "./authoritativeCreation";
import { resolveConnectionSourceCandidates } from "./topologyConnectionSources";
import {
  connectionBackingKind,
  endpointsCompatible,
  type UnifiedConnectionEndpoint,
} from "./topologyEndpointCompatibility";

const endpoint = (
  kind: UnifiedConnectionEndpoint["kind"],
  resourceId: string,
  port?: string,
): UnifiedConnectionEndpoint => ({
  kind,
  laboratoryId: "lab",
  resourceId,
  portId: kind === "node_interface" ? port : undefined,
  portName: kind === "network_object_port" ? port : undefined,
  displayName: `${resourceId}:${port || "access"}`,
  capabilities: [],
  availability: "free",
});

describe("TopologyWorkspace create drawer state", () => {
  it("opens one selecting drawer from the toolbar", () => {
    expect(openTopologyCreateDrawer()).toEqual({
      open: true,
      selection: undefined,
    });
  });

  it("opens the same drawer with a palette preselection", () => {
    expect(
      openTopologyCreateDrawer({
        kind: "pc",
        name: "PC",
        networkObjectKind: "pc",
      }),
    ).toMatchObject({
      open: true,
      selection: { networkObjectKind: "pc" },
    });
  });

  it("captures inspector, selection and focus without sharing mutable arrays", () => {
    const selectedIds = ["node-1"];
    const snapshot = captureTopologyCreateWorkspace({
      inspector: { collapsed: false, size: 360 },
      selectedIds,
      selectedType: "node",
      focusedResourceId: "node-1",
      activeElement: null,
    });
    selectedIds.push("node-2");
    expect(snapshot).toEqual({
      inspector: { collapsed: false, size: 360 },
      selectedIds: ["node-1"],
      selectedType: "node",
      focusedResourceId: "node-1",
      activeElement: null,
    });
  });

  it("consumes one authoritative creation without a predicted placement write", () => {
    const calls: string[] = [];
    const value = {
      networkObject: {
        id: "pc-1",
        laboratory_id: "lab",
        name: "PC 1",
        kind: "pc" as const,
        revision: 1,
        desired_state: "active",
        observed_state: "provisioning",
        config: {},
      },
      placement_assignment: {
        placement: {
          laboratory_id: "lab",
          resource_id: "pc-1",
          resource_type: "network_object" as const,
          x: 300,
          y: 220,
          revision: 1,
        },
        assigned_center: { x: 300, y: 220 },
        adjusted: true,
        reason: "collision_avoided" as const,
        footprint_class: "network-object-standard" as const,
        algorithm_version: 1,
      },
      laboratory_revision: 5,
    };
    applyAuthoritativeCreation(value, {
      merge: (creation) => calls.push(`merge:${creation.laboratory_revision}`),
      select: (id, type) => calls.push(`select:${type}:${id}`),
      focus: (id) => calls.push(`focus:${id}`),
      announce: (message) => calls.push(`announce:${message}`),
    });
    expect(calls).toEqual([
      "merge:5",
      "select:network_object:pc-1",
      "focus:pc-1",
      "announce:已创建 PC 1，为避免重叠已自动放置到附近空白位置。",
    ]);
  });
});

describe("TopologyWorkspace unified connection admission", () => {
  it.each([
    [
      endpoint("node_interface", "node-a", "if-a"),
      endpoint("node_interface", "node-b", "if-b"),
      "link",
    ],
    [
      endpoint("node_interface", "node-a", "if-a"),
      endpoint("network_object_port", "pc-a", "eth0"),
      "network_attachment",
    ],
    [
      endpoint("network_object_port", "pc-a", "eth0"),
      endpoint("network_object_port", "pc-b", "eth0"),
      "network_object_link",
    ],
    [
      endpoint("node_interface", "node-a", "if-a"),
      endpoint("network_object_access", "bridge-a"),
      "network_attachment",
    ],
    [
      endpoint("node_interface", "node-a", "if-a"),
      endpoint("network_object_access", "nat-a"),
      "network_attachment",
    ],
  ])(
    "maps a supported endpoint pair to %s without optimistic backing",
    (source, target, backing) => {
      expect(endpointsCompatible(source, target)).toMatchObject({
        compatible: true,
        backingKind: backing,
      });
      expect(connectionBackingKind(source, target)).toBe(backing);
    },
  );

  it("rejects an occupied target so conflict handling can refresh authoritative state", () => {
    const source = endpoint("node_interface", "node-a", "if-a");
    const target = {
      ...endpoint("network_object_port", "switch-a", "eth0"),
      availability: "occupied" as const,
    };
    expect(endpointsCompatible(source, target)).toMatchObject({
      compatible: false,
      reason: "endpoint_occupied",
    });
  });
});

describe("TopologyWorkspace unified plus source resolution", () => {
  it("auto-selects one source, opens a chooser for many, and exposes logical access", () => {
    const resources = {
      laboratoryId: "lab",
      nodes: [{ id: "node-a", name: "Node A", kind: "qemu" as const }],
      interfaces: [
        { id: "if-a", node_id: "node-a", name: "eth0" },
        { id: "if-b", node_id: "node-a", name: "eth1" },
      ],
      networkObjects: [
        {
          id: "bridge-a",
          name: "Bridge A",
          kind: "bridge" as const,
          config: {},
        },
      ],
      occupiedObjectPorts: new Set<string>(),
    };
    expect(resolveConnectionSourceCandidates("node-a", resources)).toHaveLength(
      2,
    );
    expect(resolveConnectionSourceCandidates("bridge-a", resources)).toEqual([
      expect.objectContaining({
        kind: "network_object_access",
        resourceId: "bridge-a",
      }),
    ]);
  });
});

const link: NetworkObjectLink = {
  id: "object-link-1",
  laboratory_id: "lab",
  object_a_id: "switch-a",
  port_a_name: "swp1",
  object_b_id: "switch-b",
  port_b_name: "swp1",
  revision: 3,
  desired_state: "connected",
  observed_state: "connected",
};

const envelope = {
  network_object_link: { ...link, observed_state: "disconnecting" },
  task: {
    id: "task-1",
    kind: "network_object_link.delete",
    resource_type: "network_object_link",
    resource_id: link.id,
    state: "queued",
    progress_current: 0,
    progress_total: 1,
    created_at: "2026-08-03T00:00:00Z",
  },
} satisfies NetworkObjectLinkTaskEnvelope;

describe("TopologyWorkspace object-link deletion", () => {
  it("hides and clears before submitting, then records the durable task", async () => {
    const calls: string[] = [];
    await runObjectLinkDeletion(link, {
      hide: () => calls.push("hide"),
      clearSelection: () => calls.push("clear"),
      submit: vi.fn(async () => {
        calls.push("submit");
        return envelope;
      }),
      recordTask: () => calls.push("task"),
      unhide: vi.fn(),
      reload: vi.fn(),
    });
    expect(calls).toEqual(["hide", "clear", "submit", "task"]);
  });

  it("restores and reloads a link when submission fails", async () => {
    const unhide = vi.fn();
    const reload = vi.fn();
    await expect(
      runObjectLinkDeletion(link, {
        hide: vi.fn(),
        clearSelection: vi.fn(),
        submit: vi.fn(async () => {
          throw new Error("revision conflict");
        }),
        recordTask: vi.fn(),
        unhide,
        reload,
      }),
    ).rejects.toThrow("revision conflict");
    expect(unhide).toHaveBeenCalledWith(link.id);
    expect(reload).toHaveBeenCalledOnce();
  });
});
