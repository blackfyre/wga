import { registerItineraryKeyboard } from "./itinerary";
import {
	captureTestMessage,
	initialiseSentry,
	loadSentryConfiguration,
} from "./sentry";

const sentryReady = initialiseSentry(loadSentryConfiguration(document));
// Bind itinerary keyboard navigation synchronously so Arrow keys respond the
// instant a directly-loaded viewer appears, without waiting for the async
// bootstrap chunk. The binder is idempotent and a no-op on pages without a
// viewer, so the bootstrap's later (and HTMX swap) calls stay single-bound.
registerItineraryKeyboard();
if (window.location.pathname === "/sentry-test") {
	if (sentryReady) {
		captureTestMessage();
	}
} else {
	void import("./bootstrap");
}
