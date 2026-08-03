# NetLab Wireshark Helper

The browser uses a loopback-only helper to open a live NetLab capture in the Wireshark installed on the user's computer.

## Install

Download the helper for the client operating system from the Capture panel, install Wireshark, then run one of the following commands while NetLab is in use.

The helper packages downloaded from this NetLab server already trust `http://10.72.1.159:18082`. On Windows, the downloaded `.exe` can therefore be started directly by double-clicking it and should remain running while captures are opened. The explicit `-allow-origin` commands below remain available when overriding that default.

Linux:

```bash
chmod +x netlab-wireshark-helper-linux-amd64
./netlab-wireshark-helper-linux-amd64 -allow-origin http://10.72.1.159:18082
```

Windows PowerShell:

```powershell
.\netlab-wireshark-helper-windows-amd64.exe -allow-origin http://10.72.1.159:18082
```

macOS Intel:

```bash
chmod +x netlab-wireshark-helper-darwin-amd64
./netlab-wireshark-helper-darwin-amd64 -allow-origin http://10.72.1.159:18082
```

macOS Apple Silicon:

```bash
chmod +x netlab-wireshark-helper-darwin-arm64
./netlab-wireshark-helper-darwin-arm64 -allow-origin http://10.72.1.159:18082
```

The helper listens only on `127.0.0.1:38765`. It accepts launch requests only from the configured NetLab origin and only for NetLab capture-stream URLs on that same origin.

## Use

1. Start the helper on the computer running the browser.
2. Right-click a running node and choose `Capture interface…`.
3. Select an interface and start the capture.
4. Select `Open Wireshark`.

If the helper cannot be reached, trusts another NetLab origin, cannot find Wireshark, or cannot open the capture stream, the Capture panel displays an installation or launch error instead of silently failing.
