import * as Sentry from "@sentry/browser";

export type SentryConfiguration = {
	dsn: string;
	environment: string;
};

const scrubURL = (value: string): string => {
	try {
		const isRelative =
			!value.startsWith("//") && !/^[a-zA-Z][a-zA-Z\d+.-]*:/.test(value);
		const url = new URL(value, "https://wga.invalid");
		url.search = "";
		url.hash = "";
		if (isRelative) {
			return url.pathname;
		}
		return url.toString();
	} catch {
		return value;
	}
};

const scrubBreadcrumb = (breadcrumb: Sentry.Breadcrumb): Sentry.Breadcrumb => {
	if (!breadcrumb.data) {
		return breadcrumb;
	}

	let data = { ...breadcrumb.data };
	if (breadcrumb.category === "console") {
		data = Object.fromEntries(
			Object.entries(data).filter(([key]) => key !== "arguments"),
		);
	}
	for (const key of ["url", "from", "to"]) {
		if (typeof data[key] === "string") {
			data[key] = scrubURL(data[key]);
		}
	}

	return { ...breadcrumb, data };
};

export const scrubSentryEvent = (event: Sentry.Event): Sentry.Event => ({
	...event,
	request: event.request?.url
		? { ...event.request, url: scrubURL(event.request.url) }
		: event.request,
	breadcrumbs: event.breadcrumbs?.map(scrubBreadcrumb),
});

type MetadataDocument = {
	querySelector: (selector: string) => {
		getAttribute: (name: string) => string | null;
	} | null;
};

export const loadSentryConfiguration = (
	document: MetadataDocument,
): SentryConfiguration => ({
	dsn:
		document
			.querySelector('meta[name="sentry-dsn"]')
			?.getAttribute("content") ?? "",
	environment:
		document
			.querySelector('meta[name="sentry-environment"]')
			?.getAttribute("content") ?? "",
});

export const initialiseSentry = (
	configuration: SentryConfiguration,
	initialise: typeof Sentry.init = Sentry.init,
): boolean => {
	if (configuration.dsn === "") {
		return false;
	}

	try {
		const client = initialise({
			dsn: configuration.dsn,
			environment: configuration.environment,
			beforeBreadcrumb: scrubBreadcrumb,
			beforeSend: scrubSentryEvent,
		});
		if (!client) {
			console.error("Sentry browser initialisation failed");
			return false;
		}
		return true;
	} catch (error) {
		console.error("Sentry browser initialisation failed", error);
		return false;
	}
};

export const captureTestMessage = (
	capture: typeof Sentry.captureMessage = Sentry.captureMessage,
) => {
	capture("It works!");
};
