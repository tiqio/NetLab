import { expect, test } from "./fixtures/acceptanceFixture";

test("template and automation views render authoritative service data", async ({
  page,
  automation,
}) => {
  const [templatesResponse, imagesResponse] = await Promise.all([
    automation.get("/api/v1/templates"),
    automation.get("/api/v1/images"),
  ]);
  const templates = (await templatesResponse.json()) || [];
  const images = (await imagesResponse.json()) || [];
  await page.goto("/templates");
  await expect(page.getByText(/镜像来源/)).toBeVisible();
  if (templates.length)
    await expect(
      page.getByRole("heading", {
        name: templates[0].display_name,
        exact: true,
      }),
    ).toBeVisible();
  const referencedImageId = templates
    .flatMap(
      (template: { versions?: Array<{ image_version_id?: string }> }) =>
        template.versions || [],
    )
    .find((version) => version.image_version_id)?.image_version_id;
  const referencedImage = images.find(
    (image: { id: string }) => image.id === referencedImageId,
  );
  if (referencedImage)
    await expect(
      page.getByText(referencedImage.digest, { exact: true }),
    ).toBeVisible();
  await page.goto("/automation");
  await expect(page.getByText("REST 与 MCP 能力一致性")).toBeVisible();
  const refresh = page
    .getByRole("button", { name: "刷新", exact: true })
    .first();
  await refresh.click();
  await expect(page.getByRole("heading", { name: "审计" })).toBeVisible();
});
