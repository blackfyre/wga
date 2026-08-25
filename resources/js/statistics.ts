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

// Rams theme tokens are read from the root element at chart-build time so the
// charts follow the active light/dark theme, including theme changes while the
// page is open.
const seriesTones = [
	"--wga-series-0",
	"--wga-series-1",
	"--wga-series-2",
	"--wga-series-3",
	"--wga-series-4",
	"--wga-series-5",
	"--wga-series-6",
];

function resolveTone(name: string): string {
	const value = getComputedStyle(document.documentElement)
		.getPropertyValue(name)
		.trim();
	return value || "#999999";
}

const chartText = (): string => resolveTone("--wga-text");
const chartMutedText = (): string => resolveTone("--wga-muted");
const chartRule = (): string => resolveTone("--wga-rule");

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

const schoolTones: Record<string, string> = {
	Italian: "--wga-series-0",
	French: "--wga-series-1",
	Dutch: "--wga-series-2",
	Flemish: "--wga-series-3",
	German: "--wga-series-4",
	English: "--wga-series-5",
	Spanish: "--wga-series-6",
	Other: "--wga-fill-line",
};

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

	const colors = data.map((_, i) =>
		resolveTone(seriesTones[i % seriesTones.length]),
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
					backgroundColor: colors,
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

	const datasets = orderedSchools.map((school) => ({
		label: school,
		data: periods.map((period) => {
			const row = rows.find(
				(r) => r.period_start === period && r.school === school,
			);
			return row ? row.count : 0;
		}),
		backgroundColor: resolveTone(schoolTones[school] ?? "--wga-fill-line"),
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
				legend: {
					position: "bottom",
					labels: {
						boxWidth: 10,
						color: chartMutedText(),
						font: {
							family: "ui-monospace, SF Mono, Menlo, monospace",
							size: 11,
						},
					},
				},
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
}

let themeObserver: MutationObserver | null = null;

// Rebuilds the charts whenever the active theme changes so their colours stay
// in sync with the Rams light/dark tokens while the page is open.
function watchThemeChanges(): void {
	if (themeObserver) {
		return;
	}
	themeObserver = new MutationObserver(() => {
		initStatisticsChart();
	});
	themeObserver.observe(document.documentElement, {
		attributes: true,
		attributeFilter: ["data-theme"],
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
