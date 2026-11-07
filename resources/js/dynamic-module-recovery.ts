const recoveryKey = "wga.dynamic-module-recovery";
const entryPath = "/assets/js/app.js";

export const recoverDynamicModule = (
	storage: Storage,
	url: string,
	reload: () => void,
): boolean => {
	if (storage.getItem(recoveryKey) === url) {
		return false;
	}

	storage.setItem(recoveryKey, url);
	reload();
	return true;
};

export const clearDynamicModuleRecovery = (storage: Storage): void => {
	storage.removeItem(recoveryKey);
};

// The page that failed can be an edge-cached document from the previous
// release whose entry URL the browser still holds for hours. Reloading alone
// would reuse that cached entry, so re-request it past the browser cache
// first: the origin serves the current entry under any version query.
export const refreshEntryModule = async (
	document: Document,
	fetcher: typeof fetch,
): Promise<void> => {
	const entry = Array.from(
		document.querySelectorAll<HTMLScriptElement>('script[type="module"][src]'),
	).find((script) => new URL(script.src).pathname === entryPath);
	if (!entry) {
		return;
	}
	try {
		const response = await fetcher(entry.src, {
			cache: "reload",
			credentials: "same-origin",
		});
		await response.arrayBuffer();
	} catch {
		// Reload regardless: the bounded retry state covers a failed refresh.
	}
};

export const renderDynamicModuleRetry = (document: Document): void => {
	const notice = document.createElement("div");
	notice.setAttribute("role", "alert");
	notice.className =
		"wga-alert fixed inset-x-4 top-4 z-100 border-wga-error bg-wga-bg text-wga-ink shadow-lg";
	notice.textContent = "The page assets changed while this page was open. ";
	const retry = document.createElement("button");
	retry.type = "button";
	retry.className = "wga-link font-semibold";
	retry.textContent = "Reload page";
	retry.addEventListener("click", () => window.location.reload());
	notice.append(retry);
	document.body.append(notice);
};
