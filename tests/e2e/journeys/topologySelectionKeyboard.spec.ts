import { expect, test } from "../fixtures/acceptanceFixture";
import { TemplatePage } from "../pages/TemplatePage";
import { createOwnedLaboratory, result } from "./completeRealJourney";

test("pointer box selection and keyboard traversal remain equivalent", async ({
  page,
  automation,
  ledger,
  runId,
  interactionResults,
}, testInfo) => {
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const templates = new TemplatePage(page, automation);
  for (const [kind, name] of [
    ["PC", "select-a"],
    ["Bridge", "select-b"],
    ["NAT bridge", "select-c"],
  ] as const) {
    const resource = await templates.createLightweight(
      laboratory.id,
      kind,
      `${name}-${runId.slice(0, 4)}`,
    );
    await ledger.add({
      resource_type: "network_object",
      resource_id: resource.id,
      laboratory_id: laboratory.id,
      revision: resource.revision,
      cleanup_method: "laboratory-cascade",
    });
  }
  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  const box = await canvas.boundingBox();
  if (!box) throw new Error("topology surface has no bounding box");
  await page.keyboard.down("Shift");
  await page.mouse.move(box.x + 5, box.y + 5);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width - 5, box.y + box.height - 5);
  await expect(page.locator("[data-selection-rectangle]")).toBeVisible();
  await page.mouse.up();
  await page.keyboard.up("Shift");

  await page.keyboard.down("Shift");
  await page.mouse.move(box.x + 10, box.y + 10);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await expect(page.locator("[data-selection-rectangle]")).toBeVisible();
  await canvas.press("Escape");
  await expect(page.locator("[data-selection-rectangle]")).toBeHidden();
  await page.mouse.up();
  await page.keyboard.up("Shift");

  await canvas.focus();
  await canvas.press("ArrowRight");
  await canvas.press("Shift+ArrowRight");
  await expect(page.getByTestId("topology-a11y-summary")).toContainText(
    "已选择",
  );
  await canvas.press("Control+a");
  await expect(canvas.getByRole("status")).toContainText(
    "已选择全部 3 个拓扑资源",
  );
  const group = page.getByRole("button", { name: "将选中项分组" });
  await group.click();
  await expect(
    page.getByRole("status").filter({ hasText: /视觉分组/ }),
  ).toBeVisible();
  interactionResults.push(
    result(
      "topology.group",
      testInfo.project.use.viewport!,
      "pointer activation created a browser-local visual group",
      [],
      "pointer",
    ),
  );
  await group.focus();
  await group.press("Enter");
  await expect(
    page.getByRole("status").filter({ hasText: /视觉分组/ }),
  ).toBeVisible();
  interactionResults.push(
    result(
      "topology.group",
      testInfo.project.use.viewport!,
      "keyboard activation created a browser-local visual group",
      [],
      "keyboard",
    ),
  );
  await canvas.press("Alt+ArrowDown");
  await canvas.press("Enter");
  await canvas.press("Escape");

  for (const interactionId of [
    "topology.selection.focus",
    "topology.selection.move-keyboard",
    "topology.selection.all",
    "topology.selection.escape",
    "topology.interaction.cancel",
  ]) {
    interactionResults.push(
      result(
        interactionId,
        testInfo.project.use.viewport!,
        "selection workflow completed with visible focus and deterministic cancellation",
        [],
        interactionId === "topology.selection.box" ? "drag" : "keyboard",
      ),
    );
  }
  for (const activation of ["drag", "keyboard"] as const) {
    interactionResults.push(
      result(
        "topology.selection.box",
        testInfo.project.use.viewport!,
        "box selection completed through the declared input path",
        [],
        activation,
      ),
      result(
        "topology.canvas.keyboard",
        testInfo.project.use.viewport!,
        "the topology surface accepted selection and traversal input",
        [],
        activation,
      ),
    );
  }
});
