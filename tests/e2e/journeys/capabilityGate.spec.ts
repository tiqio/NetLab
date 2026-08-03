import { expect, test } from "../fixtures/acceptanceFixture";
import { optionalEnvironmentDecision } from "../fixtures/preflight";

test("supported capabilities run and only desktop Wireshark may skip", async ({
  environment,
}) => {
  expect(
    environment.capability_decisions
      .filter((item) => item.class === "product-supported")
      .every(
        (item) =>
          item.decision === "run" ||
          environment.target_kind === "local-disposable",
      ),
  ).toBe(true);
  expect(
    optionalEnvironmentDecision("desktop-wireshark", false, "headless runner"),
  ).toMatchObject({ class: "environment-optional", decision: "skip" });
});
