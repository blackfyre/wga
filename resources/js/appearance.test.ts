import { describe, expect, test } from "bun:test";
import {
	effectiveScheme,
	isDarkOnlyPalette,
	PALETTE_NAMES,
	parsePalette,
	parseScheme,
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

	test("forces only Baroque and Tokyo to a dark effective scheme", () => {
		expect(isDarkOnlyPalette("baroque")).toBeTrue();
		expect(isDarkOnlyPalette("tokyo")).toBeTrue();
		expect(isDarkOnlyPalette("bone")).toBeFalse();
		expect(effectiveScheme("light", "baroque")).toBe("dark");
		expect(effectiveScheme("light", "tokyo")).toBe("dark");
		expect(effectiveScheme("light", "bone")).toBe("light");
	});
});
