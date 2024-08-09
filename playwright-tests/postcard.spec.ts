import { test, expect } from "@playwright/test";

test("send postcard", async ({ page }) => {
  await page.goto("/artists/koedijck-isaack-3ed9e200b9e8252");
  await page.getByRole("link", { name: "Send postcard" }).click();

  await expect(page.locator("#d")).toBeVisible();

  await expect(page.locator("#d")).toHaveText(/Write a postcard/);

  await page.locator("[name='sender_name']").fill("Playwright Tester");
  await page
    .locator("[name='sender_email']")
    .fill("playwright.tester@local.host"); // this is the postcard sender's email
  await page
    .locator("[name='recipients[]']")
    .fill("playwright.tester@local.host"); // this is the postcard recipient's email
  await page.locator("trix-editor").fill("I am testing your site.");

  await page.getByRole("button", { name: "Send postcard" }).click();

  await expect(page.locator(".toast")).toHaveText(
    /Thank you! Your postcard has been queued for sending!/,
  );

  const mailpitUrl = process.env.MAILPIT_URL;
  if (!mailpitUrl) {
    throw new Error("MAILPIT_URL environment variable is not set.");
  }
  await page.goto(mailpitUrl);

  try {
    await page
      .getByRole("link", { name: "WGA playwright.tester@local." })
      .nth(0)
      .click({
        timeout: 90 * 60 * 1000,
      });
  } catch (e) {
    console.error("Error: ", e);
    page.reload();
    await page
      .getByRole("link", { name: "WGA playwright.tester@local." })
      .nth(0)
      .click({
        timeout: 90 * 60 * 1000,
      });
  }

  const postcardLink = await page
    .frameLocator("#preview-html")
    .getByRole("link", { name: "Pickup my Postcard!" })
    .getAttribute("href", { timeout: 90000 });

  await page.getByRole("button", { name: /Delete/ }).click();

  if (!postcardLink) {
    throw new Error("Postcard link not found");
  }

  console.log("Postcard link: ", postcardLink);

  await page.goto(postcardLink);
  await expect(page.locator("#mc-area")).toContainText([
    "I am testing your site",
  ]);
});
