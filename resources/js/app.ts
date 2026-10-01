import {
	clearDynamicModuleRecovery,
	recoverDynamicModule,
	renderDynamicModuleRetry,
} from "./dynamic-module-recovery";
import { registerItineraryKeyboard } from "./itinerary";
import { registerSearchPrefs } from "./search-prefs";
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
// Apply the remembered artwork search actions setting before the bootstrap
// chunk loads, so a visitor without a cookie sees no late toggle change.
registerSearchPrefs();
if (window.location.pathname === "/sentry-test") {
	if (sentryReady) {
		captureTestMessage();
	}
} else {
	void import("./bootstrap")
		.then(() => clearDynamicModuleRecovery(window.sessionStorage))
		.catch(() => {
			if (
				!recoverDynamicModule(window.sessionStorage, window.location.href, () =>
					window.location.reload(),
				)
			) {
				renderDynamicModuleRetry(document);
			}
		});
}
