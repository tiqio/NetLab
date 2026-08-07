import { describe, expect, it } from "vitest";
import { topologySymbol } from "./topologySymbols";

function decodeSymbol(symbol: string) {
  return decodeURIComponent(
    symbol.replace("image://data:image/svg+xml;charset=UTF-8,", ""),
  );
}

describe("topology symbols", () => {
  it.each([
    ["qemu", "#2563eb"],
    ["docker", "#0f766e"],
    ["pc", "#7c3aed"],
    ["switch_l3", "#7c3aed"],
    ["bridge", "#c2410c"],
    ["nat_bridge", "#c2410c"],
  ])("renders %s with its light-theme type fill", (kind, fill) => {
    expect(
      decodeSymbol(
        topologySymbol(kind, { theme: "light", observedState: "stopped" }),
      ),
    ).toContain(`fill="${fill}"`);
  });

  it("uses runtime state, selection, and traffic as the icon outline", () => {
    expect(
      decodeSymbol(
        topologySymbol("qemu", {
          theme: "dark",
          observedState: "running",
        }),
      ),
    ).toContain('stroke="#22c55e"');
    expect(
      decodeSymbol(
        topologySymbol("docker", {
          theme: "light",
          observedState: "failed",
        }),
      ),
    ).toContain('stroke="#dc2626"');
    expect(
      decodeSymbol(
        topologySymbol("pc", {
          theme: "light",
          observedState: "stopped",
          selected: true,
        }),
      ),
    ).toContain('stroke="#0f172a"');
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
