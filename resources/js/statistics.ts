import {
	ArcElement,
	BarController,
	BarElement,
	CategoryScale,
	Chart,
	DoughnutController,
	Legend,
	LinearScale,
	Tooltip,
} from "chart.js";
import logger from "./logger";

Chart.register(
	DoughnutController,
	BarController,
	ArcElement,
	BarElement,
	CategoryScale,
	LinearScale,
	Tooltip,
	Legend,
);

// The server-rendered keys are the single source of series colour: each key
// swatch carries the token it paints (data-series-token) and whether it is a
// hatch (data-series-fill). Tokens are resolved against the root element at
// chart-build time, so the charts follow the active palette and theme,
// including changes while the page is open.
const fallbackTones = [
	"--wga-series-0",
	"--wga-series-1",
	"--wga-series-2",
	"--wga-series-3",
	"--wga-series-4",
	"--wga-series-5",
	"--wga-series-6",
];

type SeriesFill = { token: string; hatch: boolean };

function resolveTone(name: string): string {
	const value = getComputedStyle(document.documentElement)
		.getPropertyValue(name)
		.trim();
	return value || "#999999";
}

const chartText = (): string => resolveTone("--wga-text");
const chartMutedText = (): string => resolveTone("--wga-muted");
const chartRule = (): string => resolveTone("--wga-rule");

function readSwatch(element: Element | null | undefined): SeriesFill | null {
	const token = element?.getAttribute("data-series-token");
	if (!token) return null;
	return {
		token,
		hatch: element?.getAttribute("data-series-fill") === "hatch",
	};
}

function fallbackFill(index: number): SeriesFill {
	return { token: fallbackTones[index % fallbackTones.length], hatch: false };
}

// hatchPattern mirrors the key's 135° hatch: rising diagonal strokes 2px
// wide, about 5px apart, on a transparent ground, so "Other" never reads as
// one of the tones. A 7px tile puts the diagonals 7/√2 ≈ 5px apart.
const hatchTile = 7;

function hatchPattern(colour: string): CanvasPattern | string {
	const tile = document.createElement("canvas");
	tile.width = hatchTile;
	tile.height = hatchTile;
	const context = tile.getContext("2d");
	if (!context) return colour;
	context.strokeStyle = colour;
	context.lineWidth = 2;
	context.beginPath();
	for (const offset of [-hatchTile, 0, hatchTile]) {
		context.moveTo(offset, hatchTile);
		context.lineTo(offset + hatchTile, 0);
	}
	context.stroke();
	return context.createPattern(tile, "repeat") ?? colour;
}

type ResolvedFill = { paint: CanvasPattern | string; label: string };

function resolveFill(fill: SeriesFill): ResolvedFill {
	const colour = resolveTone(fill.token);
	if (fill.hatch) {
		return { paint: hatchPattern(colour), label: `hatch:${colour}` };
	}
	return { paint: colour, label: colour };
}

// recordSeriesColours exposes the colours each chart was drawn with, so
// browser tests can compare them with the key swatches without sampling
// canvas pixels.
function recordSeriesColours(
	canvas: HTMLCanvasElement,
	fills: ResolvedFill[],
): void {
	canvas.dataset.seriesColours = JSON.stringify(fills.map((f) => f.label));
}

function chartAnimation(): false | undefined {
	if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
		return false;
	}
	return undefined;
}

// animationLabel records the resolved animation mode on the canvas so browser
// tests can assert the non-animated reduced-motion path without relying on
// Chart.js internals.
function animationLabel(animation: false | undefined): string {
	if (animation === false) {
		return "none";
	}
	return "animated";
}

// Preferred display order — Other always last
const schoolOrder = [
	"Italian",
	"French",
	"Dutch",
	"Flemish",
	"German",
	"English",
	"Spanish",
	"Other",
];

type SchoolPeriodRow = { period_start: number; school: string; count: number };

const chartInstances: Record<string, Chart> = {};
const chartIDs = [
	"art-form-chart",
	"artworks-by-period-chart",
	"artists-by-period-chart",
];

function readJson(elementId: string): unknown[] {
	const el = document.getElementById(elementId);
	if (!el) return [];
	const raw = el.getAttribute("data-json") || "[]";
	try {
		return JSON.parse(raw);
	} catch (e) {
		logger.error(`Failed to parse data from #${elementId}`, e);
		return [];
	}
}

function destroyChart(id: string): void {
	if (chartInstances[id]) {
		chartInstances[id].destroy();
		delete chartInstances[id];
	}
}

export function destroyStatisticsCharts(): void {
	for (const chartID of chartIDs) {
		destroyChart(chartID);
	}
}

function initDonutChart(): void {
	destroyChart("art-form-chart");

	const canvas = document.getElementById(
		"art-form-chart",
	) as HTMLCanvasElement | null;
	if (!canvas) return;

	const data = readJson("art-form-data") as { name: string; count: number }[];
	if (data.length === 0) return;

	const swatches = document.querySelectorAll(
		"#art-form-summary tbody [data-series-token]",
	);
	const fills = data.map((_, i) =>
		resolveFill(readSwatch(swatches[i]) ?? fallbackFill(i)),
	);
	const border = resolveTone("--wga-bg");
	const animation = chartAnimation();

	chartInstances["art-form-chart"] = new Chart(canvas, {
		type: "doughnut",
		data: {
			labels: data.map((d) => d.name),
			datasets: [
				{
					data: data.map((d) => d.count),
					backgroundColor: fills.map((f) => f.paint),
					borderColor: border,
					borderWidth: 1,
				},
			],
		},
		options: {
			responsive: true,
			animation,
			plugins: {
				legend: {
					display: false,
				},
				tooltip: {
					callbacks: {
						label: (ctx) => {
							const total = (ctx.dataset.data as number[]).reduce(
								(a, b) => a + b,
								0,
							);
							const pct = ((ctx.parsed / total) * 100).toFixed(1);
							return `${ctx.label}: ${ctx.parsed.toLocaleString()} (${pct}%)`;
						},
					},
				},
			},
		},
	});
	canvas.dataset.chartAnimation = animationLabel(animation);
	recordSeriesColours(canvas, fills);
}

function buildStackedBarChart(
	canvasId: string,
	dataElementId: string,
	totalLabel: string,
): void {
	destroyChart(canvasId);

	const canvas = document.getElementById(canvasId) as HTMLCanvasElement | null;
	if (!canvas) return;

	const rows = readJson(dataElementId) as SchoolPeriodRow[];
	if (rows.length === 0) return;

	const periods = [...new Set(rows.map((r) => r.period_start))].sort(
		(a, b) => a - b,
	);
	const schools = [...new Set(rows.map((r) => r.school))];
	const orderedSchools = schoolOrder.filter((s) => schools.includes(s));

	const labels = periods.map((p) => `${p}–${p + 49}`);

	const key = canvas.closest("section")?.querySelector("[data-school-key]");
	const fills = orderedSchools.map((school) => {
		const swatch = key?.querySelector(
			`[data-school="${CSS.escape(school)}"] [data-series-token]`,
		);
		return resolveFill(
			readSwatch(swatch) ?? fallbackFill(schoolOrder.indexOf(school)),
		);
	});

	const datasets = orderedSchools.map((school, index) => ({
		label: school,
		data: periods.map((period) => {
			const row = rows.find(
				(r) => r.period_start === period && r.school === school,
			);
			return row ? row.count : 0;
		}),
		backgroundColor: fills[index].paint,
		stack: "stack",
	}));

	const animation = chartAnimation();

	chartInstances[canvasId] = new Chart(canvas, {
		type: "bar",
		data: { labels, datasets },
		options: {
			responsive: true,
			aspectRatio: 2,
			animation,
			scales: {
				x: {
					stacked: true,
					border: { color: chartRule() },
					grid: { display: false },
					ticks: {
						color: chartMutedText(),
						font: {
							family: "ui-monospace, SF Mono, Menlo, monospace",
							size: 10,
						},
						maxRotation: 45,
						minRotation: 45,
					},
				},
				y: {
					stacked: true,
					border: { color: chartRule() },
					grid: { color: chartRule() },
					ticks: { color: chartMutedText() },
					title: { color: chartText(), display: true, text: totalLabel },
				},
			},
			plugins: {
				// The server-rendered school key below the canvas is the legend.
				legend: { display: false },
				tooltip: {
					callbacks: {
						footer: (items) => {
							const total = items.reduce(
								(sum, i) => sum + (i.parsed.y as number),
								0,
							);
							return `Total: ${total.toLocaleString()}`;
						},
					},
				},
			},
		},
	});
	canvas.dataset.chartAnimation = animationLabel(animation);
	recordSeriesColours(canvas, fills);
}

let themeObserver: MutationObserver | null = null;

// Rebuilds the charts whenever the active palette or theme changes so their
// colours stay in sync with the key while the page is open.
function watchThemeChanges(): void {
	if (themeObserver) {
		return;
	}
	themeObserver = new MutationObserver(() => {
		initStatisticsChart();
	});
	themeObserver.observe(document.documentElement, {
		attributes: true,
		attributeFilter: ["data-palette", "data-theme"],
	});
}

export function initStatisticsChart(): void {
	watchThemeChanges();
	requestAnimationFrame(() => {
		initDonutChart();
		buildStackedBarChart(
			"artworks-by-period-chart",
			"artworks-period-data",
			"Artworks",
		);
		buildStackedBarChart(
			"artists-by-period-chart",
			"artists-period-data",
			"Artists",
		);
	});
}
