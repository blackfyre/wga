import { expect, type Page, test } from "@playwright/test";
import {
	curatedArtist,
	relationshipArtwork,
} from "./helpers/synthetic-fixture";

// Running prose follows the design's --t-16 at 1.7 line-height in the text
// role, and prose links use the accent with a 1px underline and no change of
// weight (WGA Prototype P:494, P:2272, P:2502). At a 16px root --t-16 is 17px.
const proseFontSize = "17px";
const proseLineHeight = "28.9px";

const viewports = [390, 834, 1440] as const;
const schemes = ["light", "dark"] as const;

// The synthetic fixture has no artwork with source commentary and no static
// page with a link, so those two surfaces receive representative sanitised
// prose through a response rewrite. The artwork wrapper reproduces the
// commentary container rendered by artwork.templ (pinned by
// TestArtworkBlockContainsLongBreadcrumbAndCommentaryAtNarrowWidths).
const sampleProse =
	'<p>Sample commentary with a <a href="/glossary">linked term</a> in it.</p>';

async function rewriteHTML(
	page: Page,
	path: string,
	rewrite: (html: string) => string,
) {
	await page.route(
		(url) => url.pathname === path,
		async (route) => {
			const response = await route.fetch();
			const html = await response.text();
			const rewritten = rewrite(html);
			expect(rewritten).not.toBe(html);
			await route.fulfill({ response, body: rewritten });
		},
	);
}

const surfaces = [
	{
		name: "artist biography",
		path: curatedArtist.path,
		prose: "#biography .content",
		prepare: async (_page: Page) => {},
	},
	{
		name: "artwork commentary",
		path: relationshipArtwork.path,
		prose: ".content",
		prepare: (page: Page) =>
			rewriteHTML(page, relationshipArtwork.path, (html) =>
				html.replace(
					/<p class="[^"]*">Commentary is unavailable for this artwork\.<\/p>/,
					`<div class="content min-w-0 max-w-[620px] break-words text-(length:--t-16) leading-[1.7]">${sampleProse}</div>`,
				),
			),
	},
	{
		name: "static page",
		path: "/pages/about",
		prose: "article.content",
		prepare: (page: Page) =>
			rewriteHTML(page, "/pages/about", (html) =>
				html.replace(/(<article class="content[^"]*">)/, `$1${sampleProse}`),
			),
	},
] as const;

// Resolves a colour role to the computed rgb() string the browser reports.
async function roleColour(page: Page, role: string): Promise<string> {
	return page.evaluate((property) => {
		const probe = document.createElement("span");
		probe.style.color = `var(${property})`;
		document.body.append(probe);
		const colour = getComputedStyle(probe).color;
		probe.remove();
		return colour;
	}, role);
}

for (const scheme of schemes) {
	for (const width of viewports) {
		test.describe(`prose typography in ${scheme} at ${width}px`, () => {
			test.use({ colorScheme: scheme, viewport: { width, height: 900 } });

			for (const surface of surfaces) {
				test(`${surface.name} uses the design prose scale and link style`, async ({
					page,
				}) => {
					await surface.prepare(page);
					await page.goto(surface.path);
					await expect(page.locator("html")).toHaveAttribute(
						"data-theme",
						scheme,
					);

					const prose = page.locator(surface.prose).first();
					const paragraph = prose.locator("p").first();
					await expect(paragraph).toHaveCSS("font-size", proseFontSize);
					await expect(paragraph).toHaveCSS("line-height", proseLineHeight);
					await expect(paragraph).toHaveCSS(
						"color",
						await roleColour(page, "--wga-text"),
					);

					const link = prose.locator("a").first();
					await expect(link).toHaveCSS(
						"color",
						await roleColour(page, "--wga-accent"),
					);
					await expect(link).toHaveCSS("text-decoration-line", "underline");
					await expect(link).toHaveCSS("text-decoration-thickness", "1px");
					await expect(link).toHaveCSS("font-weight", "400");
				});
			}
		});
	}
}
