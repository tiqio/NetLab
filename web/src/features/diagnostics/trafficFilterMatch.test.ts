import { describe, expect, it } from "vitest";
import { parseTrafficFilterMatch } from "./trafficFilterMatch";

describe("parseTrafficFilterMatch", () => {
  it("maps generic ports to either TCP direction", () => {
    expect(parseTrafficFilterMatch("tcp port 443")).toEqual({
      or: [
        { protocol: "tcp", source_port: 443 },
        { protocol: "tcp", destination_port: 443 },
      ],
    });
  });

  it("accepts the canonical expression returned for a bidirectional port", () => {
    expect(
      parseTrafficFilterMatch(
        "(tcp and src port 443) or (tcp and dst port 443)",
      ),
    ).toEqual({
      or: [
        { protocol: "tcp", source_port: 443 },
        { protocol: "tcp", destination_port: 443 },
      ],
    });
  });

  it("maps addresses and directional ports", () => {
    expect(
      parseTrafficFilterMatch(
        "src host 192.0.2.1 dst net 2001:db8::/64 udp src port 53 dst port 5353",
      ),
    ).toEqual({
      source_address: "192.0.2.1",
      destination_address: "2001:db8::/64",
      protocol: "udp",
      source_port: 53,
      destination_port: 5353,
    });
  });

  it("does not treat the port keyword as a destination address", () => {
    expect(parseTrafficFilterMatch("udp dst port 19002")).toEqual({
      protocol: "udp",
      destination_port: 19002,
    });
  });

  it("rejects unsupported logical text instead of silently changing it", () => {
    expect(() =>
      parseTrafficFilterMatch("tcp port 443 or udp port 53"),
    ).toThrow("OR and NOT are not supported");
  });
});
