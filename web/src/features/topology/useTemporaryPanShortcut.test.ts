import { describe, expect, it } from "vitest";
import { useTemporaryPanShortcut } from "./useTemporaryPanShortcut";

describe("useTemporaryPanShortcut", () => {
  it("holds the temporary pan tool only while Control is pressed", () => {
    const shortcut = useTemporaryPanShortcut();

    shortcut.handleTemporaryPanKeyDown(
      new KeyboardEvent("keydown", { key: "Control" }),
    );
    expect(shortcut.temporaryPanHeld.value).toBe(true);

    shortcut.handleTemporaryPanKeyUp(
      new KeyboardEvent("keyup", { key: "Control" }),
    );
    expect(shortcut.temporaryPanHeld.value).toBe(false);
  });

  it("stays active while another Control key remains held", () => {
    const shortcut = useTemporaryPanShortcut();

    shortcut.handleTemporaryPanKeyDown(
      new KeyboardEvent("keydown", { key: "Control" }),
    );
    shortcut.handleTemporaryPanKeyUp(
      new KeyboardEvent("keyup", { key: "Control", ctrlKey: true }),
    );

    expect(shortcut.temporaryPanHeld.value).toBe(true);
  });

  it("ignores other keys and releases on workspace focus loss", () => {
    const shortcut = useTemporaryPanShortcut();

    shortcut.handleTemporaryPanKeyDown(
      new KeyboardEvent("keydown", { key: "Shift" }),
    );
    expect(shortcut.temporaryPanHeld.value).toBe(false);

    shortcut.handleTemporaryPanKeyDown(
      new KeyboardEvent("keydown", { key: "Control" }),
    );
    shortcut.releaseTemporaryPan();
    expect(shortcut.temporaryPanHeld.value).toBe(false);
  });
});
