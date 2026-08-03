import { spawn } from "node:child_process";

export interface WiresharkLaunchReceipt {
  attempted: boolean;
  executable?: string;
  exit_code?: number;
  signal?: string;
}

export async function launchWireshark(
  commandUrl: string,
): Promise<WiresharkLaunchReceipt> {
  const executable = process.env.NETLAB_ACCEPTANCE_WIRESHARK_LAUNCHER;
  if (!executable) return { attempted: false };
  return new Promise((resolve, reject) => {
    const child = spawn(executable, [commandUrl], {
      shell: false,
      stdio: "ignore",
    });
    child.once("error", reject);
    child.once("exit", (code, signal) =>
      resolve({
        attempted: true,
        executable: executable.split(/[\\/]/).pop(),
        exit_code: code ?? undefined,
        signal: signal ?? undefined,
      }),
    );
  });
}
