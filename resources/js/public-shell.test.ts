import { expect, test } from "bun:test";
import { activeNavigationPath, navigationPath } from "./public-shell";

test("maps public record aliases to their owning shell destinations", () => {
	expect(navigationPath("/")).toBe("/");
	expect(navigationPath("/artists")).toBe("/artists");
	expect(navigationPath("/artists/albrecht-durer-a1")).toBe("/artists");
	expect(navigationPath("/artist/albrecht-durer-a1")).toBe("/artists");
	expect(navigationPath("/artists/albrecht-durer-a1/melencolia-work1")).toBe(
		"/artworks",
	);
	expect(navigationPath("/artist/albrecht-durer-a1/melencolia-work1")).toBe(
		"/artworks",
	);
	expect(
		navigationPath("/artists/albrecht-durer-a1/selections/selection1"),
	).toBe("/artists");
	expect(
		navigationPath("/artist/albrecht-durer-a1/selections/selection1"),
	).toBe("/artists");
	expect(navigationPath("/artworks")).toBe("/artworks");
	expect(navigationPath("/artworks/results")).toBe("/artworks/results");
	expect(navigationPath("/artwork/melencolia-work1")).toBe(
		"/artwork/melencolia-work1",
	);
});

test("selects the same active destination for every shell navigation surface", () => {
	const destinations = ["/", "/artists", "/artworks", "/statistics"];
	expect(
		activeNavigationPath(
			"/artist/albrecht-durer-a1/melencolia-work1",
			destinations,
		),
	).toBe("/artworks");
	expect(
		activeNavigationPath(
			"/artists/albrecht-durer-a1/selections/selection1",
			destinations,
		),
	).toBe("/artists");
	expect(activeNavigationPath("/statistics", destinations)).toBe(
		"/statistics",
	);
	expect(activeNavigationPath("/", destinations)).toBe("/");
});
