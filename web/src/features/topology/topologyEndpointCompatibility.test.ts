import { describe, expect, it } from "vitest";
import {
  connectionBackingKind,
  endpointKey,
  endpointsCompatible,
  type UnifiedConnectionEndpoint,
} from "./topologyEndpointCompatibility";

const node = (
  resourceId: string,
  portId: string,
): UnifiedConnectionEndpoint => ({
  kind: "node_interface",
  laboratoryId: "lab",
  resourceId,
  portId,
  portName: "eth0",
  displayName: `${resourceId}:eth0`,
  availability: "free",
  capabilities: [],
});

const objectPort = (
  resourceId: string,
  portName: string,
): UnifiedConnectionEndpoint => ({
  kind: "network_object_port",
  laboratoryId: "lab",
  resourceId,
  portName,
  displayName: `${resourceId}:${portName}`,
  availability: "free",
  capabilities: [],
});

describe("topology endpoint compatibility", () => {
  it("maps supported endpoint pairs without caller-selected backing kinds", () => {
    expect(connectionBackingKind(node("a", "if-a"), node("b", "if-b"))).toBe(
      "link",
    );
    expect(
      connectionBackingKind(node("a", "if-a"), objectPort("sw", "eth0")),
    ).toBe("network_attachment");
    expect(
      connectionBackingKind(objectPort("a", "eth0"), objectPort("b", "eth1")),
    ).toBe("network_object_link");
  });

  it("rejects occupied, same-resource, cross-laboratory and unsupported pairs", () => {
    expect(
      endpointsCompatible(node("a", "if-a"), node("a", "if-b")),
    ).toMatchObject({
      compatible: false,
      reason: "invalid_topology",
    });
    expect(
      endpointsCompatible(node("a", "if-a"), {
        ...objectPort("b", "eth0"),
        laboratoryId: "other",
      }),
    ).toMatchObject({
      compatible: false,
      reason: "cross_laboratory_connection",
    });
    expect(
      endpointsCompatible(node("a", "if-a"), {
        ...objectPort("b", "eth0"),
        availability: "occupied",
      }),
    ).toMatchObject({ compatible: false, reason: "endpoint_occupied" });
  });

  it("builds stable keys for concrete and logical endpoints", () => {
    expect(endpointKey(node("a", "if-a"))).toBe("node_interface:if-a");
    expect(endpointKey(objectPort("sw", "eth3"))).toBe(
      "network_object_port:sw:eth3",
    );
  });
});
