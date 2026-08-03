import { describe, expect, it } from "vitest";
import { wiresharkCommand } from "./diagnostics";

describe("wiresharkCommand", () => {
  it("builds a Windows PowerShell command with executable discovery and actionable errors", () => {
    const command = wiresharkCommand(
      "capture-1",
      "http://10.72.1.159:18082",
      "windows",
    );
    expect(command).toContain("Get-Command wireshark.exe");
    expect(command).toContain("ProgramFiles}\\Wireshark\\Wireshark.exe");
    expect(command).toContain("Get-Command curl.exe");
    expect(command).toContain("Wireshark.exe was not found");
    expect(command).toContain(
      "http://10.72.1.159:18082/api/v1/captures/capture-1/stream",
    );
    expect(command).toContain("netlab-wireshark-");
    expect(command).toContain("[IO.File]::WriteAllText");
    expect(command).toContain("& $commandFile");
    expect(command).toContain("Capture stream: $stream");
    expect(command).toContain("Remove-Item -LiteralPath $commandFile");
    expect(command).not.toContain("$env:ComSpec /d /s /c");
  });

  it("builds a pipefail-enabled Unix command with macOS fallback", () => {
    const command = wiresharkCommand("capture-1", "http://netlab.test", "unix");
    expect(command).toContain("set -o pipefail");
    expect(command).toContain("command -v wireshark");
    expect(command).toContain("/Applications/Wireshark.app");
    expect(command).toContain("WIRESHARK_BIN");
  });

  it("rejects unsafe capture identifiers", () => {
    expect(() =>
      wiresharkCommand(
        "capture-1;Remove-Item",
        "http://netlab.test",
        "windows",
      ),
    ).toThrow("Invalid resource identifier");
  });
});
