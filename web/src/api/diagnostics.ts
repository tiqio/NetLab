import { api } from "./index";

function validateId(value: string) {
  if (!/^[A-Za-z0-9._:-]+$/.test(value))
    throw new Error("Invalid resource identifier");
  return value;
}

export const safeArtifactUrl = (artifactId: string) =>
  api.downloadArtifact(validateId(artifactId));
export const safeCaptureStreamUrl = (captureId: string) =>
  api.streamCapture(validateId(captureId));

export type WiresharkCommandPlatform = "windows" | "unix";

export const wiresharkCommand = (
  captureId: string,
  baseUrl = window.location.origin,
  platform: WiresharkCommandPlatform = "unix",
) => {
  const streamUrl = `${baseUrl}${safeCaptureStreamUrl(captureId)}`;
  if (platform === "windows") {
    return [
      `$stream = '${streamUrl}'`,
      "$wiresharkCommand = Get-Command wireshark.exe -ErrorAction SilentlyContinue",
      "$wiresharkCandidates = @($wiresharkCommand.Source, \"${env:ProgramFiles}\\Wireshark\\Wireshark.exe\", \"${env:ProgramFiles(x86)}\\Wireshark\\Wireshark.exe\", \"${env:LOCALAPPDATA}\\Programs\\Wireshark\\Wireshark.exe\")",
      "$wireshark = $wiresharkCandidates | Where-Object { $_ -and (Test-Path -LiteralPath $_) } | Select-Object -First 1",
      "if (-not $wireshark) { throw 'Wireshark.exe was not found in PATH, Program Files, Program Files (x86), or LOCALAPPDATA. Install Wireshark or pass its path to the NetLab helper with -wireshark.' }",
      "$curlCommand = Get-Command curl.exe -ErrorAction SilentlyContinue",
      "if (-not $curlCommand) { throw 'curl.exe was not found. Install a current Windows curl package or use the NetLab Wireshark helper.' }",
      "$commandFile = Join-Path ([IO.Path]::GetTempPath()) ('netlab-wireshark-' + [Guid]::NewGuid().ToString('N') + '.cmd')",
      "$commandBody = '@echo off' + [Environment]::NewLine + ('\"{0}\" --fail --show-error --no-buffer \"{1}\" | \"{2}\" -k -i -' -f $curlCommand.Source, $stream, $wireshark)",
      "[IO.File]::WriteAllText($commandFile, $commandBody, [Text.Encoding]::ASCII)",
      "$pipelineExitCode = $null",
      "try { Write-Host \"Opening live capture with $wireshark\"; Write-Host \"Capture stream: $stream\"; Write-Host \"curl executable: $($curlCommand.Source)\"; & $commandFile; $pipelineExitCode = $LASTEXITCODE } finally { Remove-Item -LiteralPath $commandFile -Force -ErrorAction SilentlyContinue }",
      "if ($pipelineExitCode -ne 0) { throw \"Wireshark capture pipeline failed with exit code $pipelineExitCode. The command file was parsed successfully, but curl.exe or Wireshark.exe failed. Confirm the capture is still running, open the stream URL from this computer, and check whether endpoint security blocked either executable.\" }",
    ].join("; ");
  }
  return [
    "set -o pipefail",
    "command -v curl >/dev/null 2>&1 || { echo 'curl was not found; install curl or use the NetLab helper.' >&2; exit 127; }",
    "WIRESHARK_BIN=${WIRESHARK_BIN:-$(command -v wireshark 2>/dev/null || true)}",
    "[ -n \"$WIRESHARK_BIN\" ] || [ ! -x /Applications/Wireshark.app/Contents/MacOS/Wireshark ] || WIRESHARK_BIN=/Applications/Wireshark.app/Contents/MacOS/Wireshark",
    "[ -n \"$WIRESHARK_BIN\" ] || { echo 'Wireshark was not found in PATH or /Applications. Install Wireshark or set WIRESHARK_BIN.' >&2; exit 127; }",
    `curl --fail --show-error --no-buffer '${streamUrl}' | "$WIRESHARK_BIN" -k -i -`,
  ].join("; ");
};
