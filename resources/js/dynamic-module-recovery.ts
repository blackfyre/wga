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
	notice.className = "fixed inset-x-4 top-4 z-100 border border-error bg-base-100 p-4 text-base-content shadow-lg";
	notice.textContent = "The page assets changed while this page was open. ";
	const retry = document.createElement("button");
	retry.type = "button";
	retry.className = "link font-semibold";
	retry.textContent = "Reload page";
	retry.addEventListener("click", () => window.location.reload());
	notice.append(retry);
	document.body.append(notice);
};
