import * as Sentry from "@sentry/browser";

export type SentryConfiguration = {
	dsn: string;
	environment: string;
};

const scrubURL = (value: string): string => {
	try {
		const isRelative = value.startsWith("/");
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

export const scrubSentryEvent = (event: Sentry.Event): Sentry.Event => ({
	...event,
	request: event.request?.url
		? { ...event.request, url: scrubURL(event.request.url) }
		: event.request,
	breadcrumbs: event.breadcrumbs?.map((breadcrumb) => {
		if (typeof breadcrumb.data?.url !== "string") {
			return breadcrumb;
		}

		return {
			...breadcrumb,
			data: { ...breadcrumb.data, url: scrubURL(breadcrumb.data.url) },
		};
	}),
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
