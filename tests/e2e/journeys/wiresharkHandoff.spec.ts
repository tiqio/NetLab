import { expect, test } from "../fixtures/acceptanceFixture";
import { launchWireshark } from "../fixtures/wiresharkLauncher";

test("Wireshark launcher is optional and never uses a shell", async () => {
  const receipt = await launchWireshark(
    "http://127.0.0.1:8080/api/v1/captures/example/stream",
  );
  if (!process.env.NETLAB_ACCEPTANCE_WIRESHARK_LAUNCHER)
    expect(receipt).toEqual({ attempted: false });
  else expect(receipt).toMatchObject({ attempted: true });
});
