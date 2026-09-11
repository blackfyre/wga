const recoveryKey = "wga.dynamic-module-recovery";

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
