import { expect, test } from "../fixtures/acceptanceFixture";
import { createOwnedLaboratory } from "./completeRealJourney";
import { selectLaboratoryByName } from "../pages/LaboratoryPage";

type Placement = {
  resource_id: string;
  resource_type: "node" | "network_object";
  x: number;
  y: number;
  revision: number;
};

const footprints = {
  node: { width: 128, height: 92, clearanceX: 22, clearanceY: 18 },
  network_object: {
    width: 132,
    height: 96,
    clearanceX: 22,
    clearanceY: 18,
  },
};

function assertNoPlacementOverlap(values: Placement[]) {
  for (let left = 0; left < values.length; left += 1) {
    for (let right = left + 1; right < values.length; right += 1) {
      const a = values[left];
      const b = values[right];
      const af = footprints[a.resource_type];
      const bf = footprints[b.resource_type];
      const separatedX =
        Math.abs(a.x - b.x) >=
        af.width / 2 + bf.width / 2 + Math.max(af.clearanceX, bf.clearanceX);
      const separatedY =
        Math.abs(a.y - b.y) >=
        af.height / 2 + bf.height / 2 + Math.max(af.clearanceY, bf.clearanceY);
      expect(separatedX || separatedY).toBe(true);
    }
  }
}

test("twenty mixed resources receive stable collision-free authoritative placements", async ({
  page,
  automation,
  ledger,
  runId,
}) => {
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  let revision = laboratory.revision;
  const createdNames: string[] = [];
  const placements: Placement[] = [];
  for (let index = 0; index < 20; index += 1) {
    const name = `placement-${index + 1}`;
    createdNames.push(name);
    const isNode = index % 2 === 0;
    const response = await automation.post(
      isNode
        ? `/api/v1/labs/${laboratory.id}/nodes`
        : `/api/v1/labs/${laboratory.id}/network-objects`,
      {
        headers: {
          "If-Match": String(revision),
          "Idempotency-Key": `${runId}-placement-${index}`,
        },
        data: isNode
          ? {
              name,
              kind: index % 4 === 0 ? "qemu" : "docker",
              interface_count: 2,
              placement_intent: {
                preferred_x: 0,
                preferred_y: 0,
                footprint_class: "node-standard",
              },
            }
          : {
              name,
              kind: ["pc", "bridge", "nat_bridge", "switch_l2", "switch_l3"][
                (index >> 1) % 5
              ],
              config: {},
              placement_intent: {
                preferred_x: 0,
                preferred_y: 0,
                footprint_class: "network-object-standard",
              },
            },
      },
    );
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.placement_assignment).toBeTruthy();
    expect(body.placement_assignment.assigned_center).toBeTruthy();
    if (index > 0) {
      expect(body.placement_assignment.adjusted).toBe(true);
      expect(body.placement_assignment.reason).toBe("collision_avoided");
    }
    placements.push(body.placement_assignment.placement as Placement);
    revision = body.laboratory_revision;
  }
  assertNoPlacementOverlap(placements);

  const beforeRefresh = await automation.get(`/api/v1/labs/${laboratory.id}`);
  expect(beforeRefresh.ok()).toBeTruthy();
  const beforeSnapshot = await beforeRefresh.json();
  expect(beforeSnapshot.placements).toHaveLength(20);
  const beforeCoordinates = Object.fromEntries(
    (beforeSnapshot.placements as Placement[]).map((item) => [
      item.resource_id,
      [item.x, item.y, item.revision],
    ]),
  );

  await page.goto("/");
  await selectLaboratoryByName(page, laboratory.name);
  const canvas = page.getByLabel(/拓扑画布键盘操作区/);
  await expect(canvas).toBeVisible();
  const summary = page.getByTestId("topology-a11y-summary");
  await expect(summary).toContainText(createdNames[0]);
  await expect(summary).toContainText(createdNames[19]);
  await page.reload();
  await expect(summary).toContainText(createdNames[0]);
  await expect(summary).toContainText(createdNames[19]);

  const afterRefresh = await automation.get(`/api/v1/labs/${laboratory.id}`);
  const afterSnapshot = await afterRefresh.json();
  expect(
    Object.fromEntries(
      (afterSnapshot.placements as Placement[]).map((item) => [
        item.resource_id,
        [item.x, item.y, item.revision],
      ]),
    ),
  ).toEqual(beforeCoordinates);
});
