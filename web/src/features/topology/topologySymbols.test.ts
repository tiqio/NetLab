import { describe, expect, it } from "vitest";
import { topologySymbol } from "./topologySymbols";

function decodeSymbol(symbol: string) {
  return decodeURIComponent(
    symbol.replace("image://data:image/svg+xml;charset=UTF-8,", ""),
  );
}

describe("topology symbols", () => {
  it.each([
    ["qemu", "#708096"],
    ["docker", "#6f877e"],
    ["pc", "#877b90"],
    ["switch_l3", "#877b90"],
    ["bridge", "#927b5d"],
    ["nat_bridge", "#927b5d"],
  ])("renders %s with its light-theme type fill", (kind, fill) => {
    expect(
      decodeSymbol(
        topologySymbol(kind, { theme: "light", observedState: "stopped" }),
      ),
    ).toContain(`fill="${fill}"`);
  });

  it.each([
    "qemu",
    "docker",
    "pc",
    "bridge",
    "nat_bridge",
    "switch_l2",
    "switch_l3",
  ])("renders %s SVG details dark in the light theme", (kind) => {
    const svg = decodeSymbol(
      topologySymbol(kind, { theme: "light", observedState: "stopped" }),
    );
    expect(svg).toContain("#302e29");
    expect(svg).not.toContain("#ffffff");
    expect(svg).not.toContain("#f0ece4");
  });

  it("keeps QEMU SVG details light in the dark theme", () => {
    expect(
      decodeSymbol(
        topologySymbol("qemu", {
          theme: "dark",
          observedState: "stopped",
        }),
      ),
    ).toContain("#f0ece4");
  });

  it("uses runtime state, selection, and traffic as the icon outline", () => {
    expect(
      decodeSymbol(
        topologySymbol("qemu", {
          theme: "dark",
          observedState: "running",
        }),
      ),
    ).toContain('stroke="#86a087"');
    expect(
      decodeSymbol(
        topologySymbol("docker", {
          theme: "light",
          observedState: "failed",
        }),
      ),
    ).toContain('stroke="#a75f5d"');
    expect(
      decodeSymbol(
        topologySymbol("pc", {
          theme: "light",
          observedState: "stopped",
          selected: true,
        }),
      ),
    ).toContain('stroke="#302e29"');
    expect(
      decodeSymbol(
        topologySymbol("nat_bridge", {
          theme: "dark",
          observedState: "active",
          trafficColor: "#ff00aa",
        }),
      ),
    ).toContain('stroke="#ff00aa"');
  });
});
