import {
	Chart,
	DoughnutController,
	BarController,
	ArcElement,
	BarElement,
	CategoryScale,
	LinearScale,
	Tooltip,
	Legend,
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

// Art pigment-inspired palette for the donut chart
const donutPalette = [
	"#c0392b", // Vermilion
	"#2980b9", // Ultramarine
	"#27ae60", // Viridian
	"#f39c12", // Yellow Ochre
	"#8e44ad", // Tyrian Purple
	"#16a085", // Malachite
	"#d35400", // Burnt Sienna
	"#2c3e50", // Ivory Black
	"#7f8c8d", // Payne's Grey
	"#bdc3c7", // Titanium White
	"#e74c3c", // Cadmium Red
	"#3498db", // Cerulean
];

// Fixed colors per school so they're consistent across both bar charts
const schoolColors: Record<string, string> = {
	Italian:  "#c0392b",
	French:   "#2980b9",
	Dutch:    "#f39c12",
	Flemish:  "#8e44ad",
	German:   "#27ae60",
	English:  "#16a085",
	Spanish:  "#d35400",
	Other:    "#7f8c8d",
};

// Preferred display order — Other always last
const schoolOrder = ["Italian", "French", "Dutch", "Flemish", "German", "English", "Spanish", "Other"];

type SchoolPeriodRow = { period_start: number; school: string; count: number };

const chartInstances: Record<string, Chart> = {};

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

function initDonutChart(): void {
	const canvas = document.getElementById("art-form-chart") as HTMLCanvasElement | null;
	if (!canvas) return;

	destroyChart("art-form-chart");

	const data = readJson("art-form-data") as { name: string; count: number }[];
	if (data.length === 0) return;

	const colors = data.map((_, i) => donutPalette[i % donutPalette.length]);

	chartInstances["art-form-chart"] = new Chart(canvas, {
		type: "doughnut",
		data: {
			labels: data.map((d) => d.name),
			datasets: [{
				data: data.map((d) => d.count),
				backgroundColor: colors,
				borderColor: colors.map((c) => c + "cc"),
				borderWidth: 1,
			}],
		},
		options: {
			responsive: true,
			plugins: {
				legend: { position: "bottom" },
				tooltip: {
					callbacks: {
						label: (ctx) => {
							const total = (ctx.dataset.data as number[]).reduce((a, b) => a + b, 0);
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
	const canvas = document.getElementById(canvasId) as HTMLCanvasElement | null;
	if (!canvas) return;

	destroyChart(canvasId);

	const rows = readJson(dataElementId) as SchoolPeriodRow[];
	if (rows.length === 0) return;

	const periods = [...new Set(rows.map((r) => r.period_start))].sort((a, b) => a - b);
	const schools = [...new Set(rows.map((r) => r.school))];
	const orderedSchools = schoolOrder.filter((s) => schools.includes(s));

	const labels = periods.map((p) => `${p}–${p + 49}`);

	const datasets = orderedSchools.map((school) => ({
		label: school,
		data: periods.map((period) => {
			const row = rows.find((r) => r.period_start === period && r.school === school);
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
			scales: {
				x: {
					stacked: true,
					ticks: { maxRotation: 45, minRotation: 45 },
				},
				y: {
					stacked: true,
					title: { display: true, text: totalLabel },
				},
			},
			plugins: {
				legend: { position: "bottom" },
				tooltip: {
					callbacks: {
						footer: (items) => {
							const total = items.reduce((sum, i) => sum + (i.parsed.y as number), 0);
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
		buildStackedBarChart("artworks-by-period-chart", "artworks-period-data", "Artworks");
		buildStackedBarChart("artists-by-period-chart", "artists-period-data", "Artists");
	});
}

// Self-init on DOMContentLoaded for direct page loads
document.addEventListener("DOMContentLoaded", () => {
	initStatisticsChart();
});
