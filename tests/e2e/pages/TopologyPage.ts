import { expect, type APIRequestContext, type Page } from "@playwright/test";
import { BasePage } from "./BasePage";
import { waitForCondition } from "../fixtures/waiters";

export class TopologyPage extends BasePage {
  constructor(
    page: Page,
    private readonly request: APIRequestContext,
  ) {
    super(page);
  }

  async connect(laboratoryId: string, endpointA: string, endpointB: string) {
    await this.page
      .getByRole("combobox", { name: "Endpoint A" })
      .selectOption(endpointA);
    await this.page
      .getByRole("combobox", { name: "Endpoint B" })
      .selectOption(endpointB);
    await this.page
      .getByRole("button", { name: "Connect live", exact: true })
      .click();
    return waitForCondition(
      async () => {
        const response = await this.request.get(`/api/v1/labs/${laboratoryId}`);
        const snapshot = await response.json();
        return (snapshot.links || []).find(
          (link: { endpoint_a_id: string; endpoint_b_id: string }) =>
            [link.endpoint_a_id, link.endpoint_b_id].includes(endpointA) &&
            [link.endpoint_a_id, link.endpoint_b_id].includes(endpointB),
        );
      },
      (link): link is { id: string } => Boolean(link),
      "live link",
      30_000,
    );
  }

  async openResourceDrawer() {
    await this.page.getByRole("button", { name: "添加资源" }).click();
    const drawer = this.page.getByRole("dialog", { name: "Add resource" });
    await expect(drawer).toBeVisible();
    return drawer;
  }

  async chooseDrawerResource(name: string) {
    const drawer = this.page.getByRole("dialog", { name: "Add resource" });
    await drawer
      .getByText(name, { exact: true })
      .locator("xpath=ancestor::button[1]")
      .click();
    return this.page.getByRole("dialog", {
      name: new RegExp(`Add ${name}`, "i"),
    });
  }

  async selectResourceByKeyboard(index: number, additive = false) {
    const canvas = this.page.getByLabel(/Topology canvas keyboard area/);
    await canvas.focus();
    for (let step = 0; step <= index; step += 1) {
      await canvas.press(
        additive && step === index ? "Shift+ArrowRight" : "ArrowRight",
      );
    }
    await expect(canvas.getByRole("status")).toContainText(
      /(?:node|link|network_object) /,
    );
  }

  async openSelectedInspector() {
    await this.page.getByLabel(/Topology canvas keyboard area/).press("Enter");
  }

  async openSelectedTerminal() {
    const canvas = this.page.getByLabel(/Topology canvas keyboard area/);
    await canvas.focus();
    await canvas.press("t");
    await expect(
      this.page.getByRole("navigation", { name: "Console sessions" }),
    ).toBeVisible();
  }

  async disconnectSelectedLink() {
    await this.page
      .getByRole("button", { name: "Disconnect live link" })
      .click();
    const dialog = this.page.getByRole("dialog", { name: "Disconnect link" });
    await dialog.getByRole("button", { name: "Disconnect" }).click();
  }

  async panAndZoom() {
    const canvas = this.page.getByRole("img", { name: /Topology canvas/ });
    await canvas.hover();
    await this.page.mouse.wheel(0, -300);
    const box = await canvas.boundingBox();
    if (!box) throw new Error("Topology canvas has no bounding box");
    await this.page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await this.page.mouse.down();
    await this.page.mouse.move(
      box.x + box.width / 2 + 40,
      box.y + box.height / 2 + 20,
    );
    await this.page.mouse.up();
  }
}
