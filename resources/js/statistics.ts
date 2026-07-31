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

const donutPalette = [
	"#003366",
	"#2a5580",
	"#54789a",
	"#7e9ab5",
	"#a8bccf",
	"#8a857c",
];

const schoolColors: Record<string, string> = {
	Italian: "#003366",
	French: "#1c4d80",
	Dutch: "#356a99",
	Flemish: "#5786af",
	German: "#7ba1c4",
	English: "#9fbbd6",
	Spanish: "#c0d2e4",
	Other: "#8a857c",
};

const chartText = "#1c1c1a";
const chartMutedText = "#6b6660";
const chartRule = "rgba(28, 28, 26, 0.15)";

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

	const colors = data.map((_, i) => donutPalette[i % donutPalette.length]);

	chartInstances["art-form-chart"] = new Chart(canvas, {
		type: "doughnut",
		data: {
			labels: data.map((d) => d.name),
			datasets: [
				{
					data: data.map((d) => d.count),
					backgroundColor: colors,
					borderColor: colors.map((c) => `${c}cc`),
					borderWidth: 1,
				},
			],
		},
		options: {
			responsive: true,
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
		backgroundColor: schoolColors[school] ?? "#999",
		stack: "stack",
	}));

	chartInstances[canvasId] = new Chart(canvas, {
		type: "bar",
		data: { labels, datasets },
		options: {
			responsive: true,
			maintainAspectRatio: false,
			scales: {
				x: {
					stacked: true,
					border: { color: chartRule },
					grid: { display: false },
					ticks: {
						color: chartMutedText,
						font: { family: "ui-monospace, SF Mono, Menlo, monospace", size: 10 },
						maxRotation: 45,
						minRotation: 45,
					},
				},
				y: {
					stacked: true,
					border: { color: chartRule },
					grid: { color: chartRule },
					ticks: { color: chartMutedText },
					title: { color: chartText, display: true, text: totalLabel },
				},
			},
			plugins: {
				legend: {
					position: "bottom",
					labels: {
						boxWidth: 10,
						color: chartMutedText,
						font: { family: "ui-monospace, SF Mono, Menlo, monospace", size: 11 },
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
}

export function initStatisticsChart(): void {
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
