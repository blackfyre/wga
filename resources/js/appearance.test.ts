import { describe, expect, test } from "bun:test";
import {
	PALETTE_NAMES,
	type Palette,
	type Scheme,
	effectiveScheme,
	isDarkOnlyPalette,
	parsePalette,
	parseScheme,
	resolveThemeName,
} from "./appearance";

describe("appearance preference resolution", () => {
	test("accepts current and legacy scheme values", () => {
		expect(parseScheme("light")).toBe("light");
		expect(parseScheme("dark")).toBe("dark");
		expect(parseScheme("wga_light")).toBe("light");
		expect(parseScheme("wga_dark")).toBe("dark");
		expect(parseScheme("invalid")).toBeNull();
	});

	test("accepts only the eleven stable palette keys", () => {
		expect(PALETTE_NAMES).toHaveLength(11);
		for (const palette of PALETTE_NAMES) {
			expect(parsePalette(palette)).toBe(palette);
		}
		expect(parsePalette("unknown")).toBeNull();
	});

	test("resolves every palette and scheme independently", () => {
		for (const palette of PALETTE_NAMES) {
			expect(resolveThemeName("light", palette)).toStartWith("wga-");
			expect(resolveThemeName("dark", palette)).toStartWith("wga-");
		}
		expect(resolveThemeName("dark", "classic")).toBe("wga-classic-dark");
		expect(resolveThemeName("light", "classic")).toBe("wga-classic");
	});

	test("resolves the exact server theme name for every palette and scheme", () => {
		const expected: Record<Palette, Record<Scheme, string>> = {
			bone: { light: "wga-rams", dark: "wga-rams-dark" },
			classic: { light: "wga-classic", dark: "wga-classic-dark" },
			verdigris: { light: "wga-verdigris", dark: "wga-verdigris-dark" },
			gothic: { light: "wga-gothic", dark: "wga-gothic-dark" },
			renaissance: {
				light: "wga-renaissance",
				dark: "wga-renaissance-dark",
			},
			baroque: { light: "wga-baroque", dark: "wga-baroque" },
			rococo: { light: "wga-rococo", dark: "wga-rococo-dark" },
			classical: { light: "wga-classical", dark: "wga-classical-dark" },
			impressionist: {
				light: "wga-impressionist",
				dark: "wga-impressionist-dark",
			},
			catppuccin: {
				light: "wga-catppuccin",
				dark: "wga-catppuccin-dark",
			},
			tokyo: { light: "wga-tokyo", dark: "wga-tokyo" },
		};
		for (const palette of PALETTE_NAMES) {
			for (const scheme of ["light", "dark"] as const) {
				expect(resolveThemeName(scheme, palette)).toBe(
					expected[palette][scheme],
				);
			}
		}
	});

	test("forces only Baroque and Tokyo to a dark effective scheme", () => {
		expect(isDarkOnlyPalette("baroque")).toBeTrue();
		expect(isDarkOnlyPalette("tokyo")).toBeTrue();
		expect(isDarkOnlyPalette("bone")).toBeFalse();
		expect(effectiveScheme("light", "baroque")).toBe("dark");
		expect(effectiveScheme("light", "tokyo")).toBe("dark");
		expect(effectiveScheme("light", "bone")).toBe("light");
		expect(resolveThemeName("light", "baroque")).toBe("wga-baroque");
		expect(resolveThemeName("light", "tokyo")).toBe("wga-tokyo");
	});
});
