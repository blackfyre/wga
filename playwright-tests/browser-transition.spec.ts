import { expect, test } from "@playwright/test";

test("artwork search survives a native transition capture failure", async ({
  page,
}) => {
  const errors: string[] = [];
  const searches: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (url.pathname === "/artworks" && url.searchParams.has("q")) {
      searches.push(url.searchParams.get("q")!);
    }
  });
  await page.goto("/artworks");
  await expect.poll(() => page.evaluate(() => Boolean(window.wga))).toBe(true);
  await page.evaluate(() => {
    document.body.dataset.searchSwaps = "0";
    document.addEventListener("htmx:afterSwap", (event) => {
      if ((event as CustomEvent).detail.target.id === "artwork-search") {
        document.body.dataset.searchSwaps = String(
          Number(document.body.dataset.searchSwaps) + 1,
        );
      }
    });
    // Duplicate names make the native capture reject ready, while the DOM
    // update callback must still run. Do not replace the browser API.
    for (let i = 0; i < 2; i++) {
      const element = document.createElement("div");
      element.textContent = "transition capture fixture";
      element.style.viewTransitionName = "duplicate-test-name";
      element.dataset.transitionFixture = "";
      document.body.append(element);
    }
  });
  await page.locator("#artwork-query").fill("Synthetic Artwork 01-01");
  await expect(page).toHaveURL(
    (url) => url.searchParams.get("q") === "Synthetic Artwork 01-01",
  );
  await expect(page.locator("#search-result-container")).toContainText(
    "1 WORK MATCHES",
  );
  // A second search proves the first skipped animation did not block swaps.
  await page.locator("#artwork-query").fill("Synthetic Artwork 01-02");
  await expect(page).toHaveURL(
    (url) => url.searchParams.get("q") === "Synthetic Artwork 01-02",
  );
  await expect(page.locator("#search-result-container")).toContainText(
    "1 WORK MATCHES",
  );
  await page.evaluate(
    () =>
      new Promise((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(resolve)),
      ),
  );
  expect(errors).toEqual([]);
  expect(searches).toEqual([
    "Synthetic Artwork 01-01",
    "Synthetic Artwork 01-02",
  ]);
  await expect(page.locator("body")).toHaveAttribute("data-search-swaps", "2");
});

test("artwork search works without the View Transitions API", async ({
  page,
}) => {
  await page.addInitScript(() => {
    Object.defineProperty(document, "startViewTransition", {
      value: undefined,
    });
  });
  await page.goto("/artworks");
  await expect.poll(() => page.evaluate(() => Boolean(window.wga))).toBe(true);
  await page.locator("#artwork-query").fill("Synthetic Artwork 01-01");
  await expect(page).toHaveURL(
    (url) => url.searchParams.get("q") === "Synthetic Artwork 01-01",
  );
  await expect(page.locator("#search-result-container")).toContainText(
    "1 WORK MATCHES",
  );
});

test("successful native transitions still animate", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/artworks");
  await expect.poll(() => page.evaluate(() => Boolean(window.wga))).toBe(true);
  await page.evaluate(() => {
    const start = document.startViewTransition;
    document.startViewTransition = function (...args) {
      const transition = start.apply(this, args);
      void transition.ready.then(() => {
        document.documentElement.dataset.transitionReady = "true";
      });
      void transition.finished.then(() => {
        document.documentElement.dataset.transitionFinished = "true";
      });
      return transition;
    };
  });
  await page.locator("#artwork-query").fill("Synthetic Artwork 01-01");
  await expect(page.locator("html")).toHaveAttribute(
    "data-transition-ready",
    "true",
  );
  await expect(page.locator("html")).toHaveAttribute(
    "data-transition-finished",
    "true",
  );
  await expect(page.locator("#search-result-container")).toContainText(
    "1 WORK MATCHES",
  );
  expect(errors).toEqual([]);
});

test("other ready rejections remain observable", async ({ page }) => {
  // A controlled API result isolates rejection classification from rendering.
  await page.addInitScript(() => {
    document.startViewTransition = (() => ({
      ready: Promise.reject(new TypeError("unexpected transition failure")),
      updateCallbackDone: Promise.resolve(),
      finished: Promise.resolve(),
    })) as typeof document.startViewTransition;
  });
  await page.goto("/artworks");
  await expect.poll(() => page.evaluate(() => Boolean(window.wga))).toBe(true);
  const failure = page.waitForEvent("pageerror");
  await page.evaluate(() => {
    document.startViewTransition(() => {});
  });
  expect((await failure).message).toContain("unexpected transition failure");
});

test("InvalidStateError in a native update callback remains observable", async ({
  page,
}) => {
  await page.goto("/artworks");
  await expect.poll(() => page.evaluate(() => Boolean(window.wga))).toBe(true);
  const failure = page.waitForEvent("pageerror");
  await page.evaluate(() => {
    document.startViewTransition(() => {
      throw new DOMException("update callback failed", "InvalidStateError");
    });
  });
  expect((await failure).message).toContain("update callback failed");
});

test("unrelated InvalidStateError remains observable", async ({ page }) => {
  await page.goto("/artworks");
  await expect.poll(() => page.evaluate(() => Boolean(window.wga))).toBe(true);
  const failure = page.waitForEvent("pageerror");
  await page.evaluate(() => {
    void Promise.reject(
      new DOMException("unrelated invalid state", "InvalidStateError"),
    );
  });
  expect((await failure).message).toContain("unrelated invalid state");
});
