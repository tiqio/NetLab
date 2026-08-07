import { describe, expect, it } from "vitest";
import { connectionSemanticMarkers } from "./topologyConnectionSemantics";

describe("connectionSemanticMarkers", () => {
  it("marks managed NAT uplinks without changing status", () => {
    expect(connectionSemanticMarkers("docker", "nat_bridge")).toEqual(["managed-nat-uplink"]);
  });

  it("marks shared broadcast domains and leaves point-to-point links plain", () => {
    expect(connectionSemanticMarkers("docker", "bridge")).toEqual(["shared-broadcast-domain"]);
    expect(connectionSemanticMarkers("docker", "docker")).toEqual([]);
  });
});
