import Viewer from "viewerjs";
import Trix from "trix";
import "htmx.org";
import htmx from "htmx.org";

declare global {
	interface Window {
		wga: wgaWindow;
	}
}

type HtmxValidateUrlEvent = CustomEvent<{
	sameHost: boolean;
}>;

type popUpWindow = {
	url: string;
	title: string;
	position: {
		x: number;
		y: number;
	};
	size: {
		width: number;
		height: number;
	};
	scrollbars?: boolean;
};

type wgaWindow = {
	dialog: {
		open: () => void;
		close: () => void;
	};
	window: {
		historyBack: () => void;
		close: () => void;
		openPopUp: (w: popUpWindow) => void;
	};
	music: {
		openPopUp: () => void;
	};
};

type wgaInternals = {
	els: {
		dialog: HTMLDialogElement | null;
		toastContainer: HTMLElement | null;
	};
	existingCloners: HTMLElement[];
	dialogDefaultContent: string;
	eventListeners: (() => void)[];
	func: {
		cloner: () => void;
		viewer: () => void;
		toast: (message: string, type: ToastEvent["detail"]["type"]) => void;
		artistSearchModal: () => void;
		combobox: () => void;
		init: () => void;
	};

	setup: {
		htmx: () => void;
		elements: () => void;
	};
};

type wgaComboBox = {
	input: {
		style: string;
		placeholder: string;
		type: string;
	};
	list: {
		style: string;
	};
	moreAvailable: {
		style: string;
		content: string;
	};
	noResults: {
		style: string;
		content: string;
	};
	item: {
		style: string;
	};
	options: object[];
	hooks: {
		onSelected: (v: string) => void;
	};
};

type wgaComboBoxConfig = {
	input?: {
		style?: string;
		placeholder?: string;
		type?: string;
	};
	list?: {
		style?: string;
	};
	moreAvailable?: {
		style?: string;
		content?: string;
	};
	noResults?: {
		style?: string;
		content?: string;
	};
	item?: {
		style?: string;
	};
	options?: {
		id: string;
		label: string;
	}[];
	hooks?: {
		onSelected?: (v: string) => void;
	};
};

interface ToastEvent extends Event {
	detail: {
		closeDialog: boolean;
		message: string;
		type: "info" | "alert" | "warning" | "error" | "success";
	};
}

const deepMerge = (target: object, source: object): object => {
	// Iterate over keys in the source object
	for (const key in source) {
		if (key === "__proto__" || key === "constructor") continue;
		if (Object.prototype.hasOwnProperty.call(source, key)) {
			// Check if the current key's value is an object and exists in the target
			if (typeof source[key] === "object" && source[key] !== null) {
				if (Array.isArray(source[key])) {
					// For arrays, merge (or you can decide to replace target[key] if needed)
					target[key] = (target[key] || []).concat(source[key]);
				} else {
					// For objects, initialize target[key] if it doesn't exist, then recourse
					target[key] = deepMerge(target[key] || {}, source[key]);
				}
			} else {
				// For primitives, directly assign the source value
				target[key] = source[key];
			}
		}
	}
	return target;
};

const paneToTarget = (side: string): string => {
	if (side === "left") {
		return "right";
	}

	return "left";
};

const dualNavAction = (side: string, url: string) => {
	console.log("Side", side);
	console.log("URL", url);
	console.log("Base URL", window.location.href.split("?")[0]);
	// Get the current url base
	const baseUrl = window.location.href.split("?")[0];

	const newUrl = new URL("/dual-mode", baseUrl);

	// Inherit the search parameters
	const searchParams = new URLSearchParams(window.location.search);
	searchParams.forEach((value, key) => {
		newUrl.searchParams.set(key, value);
	});

	// Set the search parameters
	newUrl.searchParams.set(side, url);

	// Create an anchor element
	htmx.ajax("get", newUrl.href, {
		target: `#${paneToTarget(side)}`,
		select: `#${paneToTarget(side)}`,
		swap: "outerHTML",
	});
};

const wgaInternal: wgaInternals = {
	els: {
		dialog: null,
		toastContainer: null,
	},
	existingCloners: [],
	dialogDefaultContent: "",
	eventListeners: [
		() => {
			// Toast event listener
			document.body.addEventListener("notification:toast", (evt) => {
				const event = evt as ToastEvent;
				if (event.detail.closeDialog) {
					wgaInternal.els.dialog?.close();
				}

				wgaInternal.func.toast(event.detail.message, event.detail.type);
			});
		},
		// () => {
		//   document.addEventListener("DOMContentLoaded", function (event) {
		//     wgaInternal.func.viewer();
		//     wgaInternal.func.cloner();
		//     wgaInternal.func.augmentSelects();
		//   });
		// },
		() => {
			// htmx event listeners
			document.body.addEventListener("htmx:load", () => {
				wgaInternal.func.viewer();
				wgaInternal.func.cloner();
				wgaInternal.func.artistSearchModal();
				wgaInternal.func.combobox();
			});

			document.body.addEventListener("htmx:swapError", (evt) => {
				console.error(evt);
				wgaInternal.func.toast(
					"An error occurred while processing your request.",
					"error",
				);
			});

			document.body.addEventListener("htmx:targetError", (evt) => {
				console.error(evt);
				wgaInternal.func.toast(
					"An error occurred while processing your request.",
					"error",
				);
			});

			document.body.addEventListener("htmx:timeout", (evt) => {
				console.error(evt);
				wgaInternal.func.toast("Request timed out!", "error");
			});

			document.body.addEventListener("htmx:validateUrl", (evt) => {
				const event = evt as HtmxValidateUrlEvent;
				// only allow requests to the current server
				if (!event.detail.sameHost) {
					evt.preventDefault();
				}
			});
		},
		() => {
			//! This is a workaround, has to be removed when htmx fixes the issue
			addEventListener("htmx:beforeHistorySave", () => {
				for (const el of document.querySelectorAll(":disabled")) {
					const e = el as HTMLInputElement;
					e.disabled = false;
				}
			});
		},
		() => {
			// Trix initialization
			document.addEventListener("trix-before-initialize", () => {
				Trix.config.toolbar.getDefaultHTML = () => {
					return `
        <div class="trix-button-row">
          <span class="trix-button-group trix-button-group--text-tools" data-trix-button-group="text-tools">
            <button type="button" class="trix-button trix-button--icon trix-button--icon-bold btn-neutral" data-trix-attribute="bold" data-trix-key="b" title="Bold" tabindex="-1">Bold</button>
            <button type="button" class="trix-button trix-button--icon trix-button--icon-italic btn-neutral" data-trix-attribute="italic" data-trix-key="i" title="Italic" tabindex="-1">Italic</button>
            <button type="button" class="trix-button trix-button--icon trix-button--icon-strike btn-neutral" data-trix-attribute="strike" title="Strike" tabindex="-1">Strike</button>
          </span>
    
          <span class="trix-button-group trix-button-group--block-tools" data-trix-button-group="block-tools">
            <button type="button" class="trix-button trix-button--icon trix-button--icon-heading-1 btn-neutral" data-trix-attribute="heading1" title="Heading 1" tabindex="-1">Heading 1</button>
            <button type="button" class="trix-button trix-button--icon trix-button--icon-quote btn-neutral" data-trix-attribute="quote" title="Quote" tabindex="-1">Quote</button>
          </span>
    
        </div>`;
				}; // Change `Trix.config` if you need
			});
		},
		() => {
			// Back to top button
			const jumpToTop = document.querySelector(".jump.back-to-top");
			if (jumpToTop) {
				jumpToTop.addEventListener("click", () => {
					window.scrollTo({ top: 0, behavior: "smooth" });
				});
			}
		},
	],
	func: {
		cloner: () => {
			// Find all the elements with data-cloner-target attribute
			const cloners = document.querySelectorAll("[data-cloner-target]");

			// Remove not seen items from `wga.existingCloners`
			wgaInternal.existingCloners = wgaInternal.existingCloners.filter((el) =>
				document.body.contains(el),
			);

			cloners.forEach((c) => {
				const cloner = c as HTMLElement;
				// Get the target element
				const target = document.querySelector(
					cloner.dataset.clonerTarget as string,
				);

				if (!target) {
					return;
				}

				// Get the target's innerHTML as the template
				const template = target.innerHTML;

				// If target not in `wga.existingCloners`

				if (!wgaInternal.existingCloners.includes(target as HTMLElement)) {
					// Add target to `wga.existingCloners`
					wgaInternal.existingCloners.push(target as HTMLElement);

					cloner.addEventListener("click", () => {
						// Append the template to the target
						target?.insertAdjacentHTML("beforeend", template);

						// Find all the elements with data-cloner-remove-me attribute
						const removeMe = target.querySelectorAll("[data-cloner-remove-me]");

						// Loop through each removeMe

						removeMe.forEach((el) => {
							const removeMe = () => {
								// Find the closest .field element
								const field = el.closest("label.input");

								// Remove the field
								field?.remove();
							};

							el.removeEventListener("click", removeMe);

							// Add click event listener
							el.addEventListener("click", removeMe);
						});
					});
				}
			});
		},
		viewer: () => {
			const elements = document.querySelectorAll("[data-viewer]");
			if (elements.length > 0) {
				elements.forEach((element) => {
					const e = element as HTMLElement;
					new Viewer(e, {
						toolbar: {
							zoomIn: 1,
							zoomOut: 1,
							oneToOne: 1,
							reset: 1,
							prev: 1,
							play: {
								show: 1,
								size: "large",
							},
							next: 1,
							rotateLeft: 1,
							rotateRight: 1,
							flipHorizontal: 0,
							flipVertical: 0,
						},
					});
				});
			}
		},
		toast: (message, type) => {
			// Define variants for color, title, and icon
			const colorVariants = {
				info: "alert alert-info cursor-pointer sm:alert-horizontal",
				alert: "alert cursor-pointer sm:alert-horizontal",
				warning: "alert alert-warning cursor-pointer sm:alert-horizontal",
				error: "alert alert-error cursor-pointer sm:alert-horizontal",
				success: "alert alert-success cursor-pointer sm:alert-horizontal",
			};

			const titleVariants = {
				info: "Info",
				alert: "Alert",
				warning: "Warning",
				error: "Error",
				success: "Success",
			};

			const iconVariants = {
				info: `<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>`,
				alert: `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="stroke-info h-6 w-6 shrink-0">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>`,
				warning: `<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>`,
				error: `<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>`,
				success: `<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>`,
			};

			// Create elements
			const toast = document.createElement("div");
			const mb = document.createElement("div");
			const title = document.createElement("h3");
			const toastMessage = document.createElement("span");

			// Set icon and content
			toast.innerHTML = iconVariants[type] || iconVariants.info;
			title.textContent = titleVariants[type] || "Notification";
			title.className = "font-bold";
			toastMessage.textContent = message;

			// Set classes and attributes
			toast.className =
				colorVariants[type] ||
				"alert alert-info cursor-pointer sm:alert-horizontal";
			toast.setAttribute("role", "alert");

			// Compose toast content
			mb.appendChild(title);
			mb.appendChild(toastMessage);
			toast.appendChild(mb);

			// Add close on click
			toast.addEventListener("click", () => {
				toast.remove();
			});

			// Show toast
			wgaInternal.els.toastContainer?.appendChild(toast);

			// Auto-remove after 5 seconds
			setTimeout(() => {
				toast.remove();
			}, 5000);
		},
		init() {
			console.info("WGA Internal Functions Initialized");
			// Run all internal functions
			wgaInternal.setup.htmx();
			wgaInternal.setup.elements();

			// Initialize components
			wgaInternal.func.combobox();

			// Run all event listeners
			wgaInternal.eventListeners.forEach((listener) => listener());
		},
		artistSearchModal() {
			const artistSearchModal = document.getElementById("artist_lookup");

			if (!artistSearchModal) {
				return;
			}

			//find .modal-box in the modal
			const modalBox = artistSearchModal.querySelector(".modal-box");

			if (!modalBox) {
				console.error("Modal box not found");
				return;
			}

			// get the contents of #artistList and parse it as json
			const artistList = document.getElementById("artistList");
			if (!artistList) {
				console.error("Artist list not found");
				return;
			}

			const artists = JSON.parse(artistList.innerHTML);

			const label = document.createElement("label");

			const searchInput = document.createElement("input");

			searchInput.type = "search";

			label.appendChild(searchInput);

			// create a table within the modal from the json which has label and url keys
			const tableContainer = document.createElement("div");
			tableContainer.className = "overflow-x-auto";
			const table = document.createElement("table");
			table.className = "table";

			tableContainer.appendChild(table);

			// artists.forEach((artist: { label: string; url: string }) => {
			//   const tr = document.createElement("tr");
			//   const td = document.createElement("td");
			//   const a = document.createElement("a");

			//   a.href = artist.url;
			//   a.textContent = artist.label;

			//   td.appendChild(a);
			//   tr.appendChild(td);
			//   table.appendChild(tr);
			// });

			// replace the contents of the dialog with the table
			modalBox.innerHTML = "";

			const modalTitle = document.createElement("h2");
			modalTitle.textContent = "Artist Lookup";
			modalBox.appendChild(modalTitle);
			modalBox.appendChild(document.createElement("hr"));

			modalBox.appendChild(label);
			modalBox.appendChild(tableContainer);
		},
		combobox: () => {
			// Initialize all combobox components
			const comboboxContainers = document.querySelectorAll("[data-combobox]");

			for (const container of comboboxContainers) {
				const comboboxId = container.getAttribute("data-combobox");
				if (!comboboxId) return;

				const displayInput = container.querySelector(
					`[data-combobox-input="${comboboxId}"]`,
				) as HTMLInputElement;
				const valueInput = container.querySelector(
					`[data-combobox-value="${comboboxId}"]`,
				) as HTMLInputElement;
				const dropdown = container.querySelector(
					`[data-combobox-dropdown="${comboboxId}"]`,
				) as HTMLElement;
				const searchInput = container.querySelector(
					`[data-combobox-search="${comboboxId}"]`,
				) as HTMLInputElement;
				const optionsList = container.querySelector(
					`[data-combobox-options="${comboboxId}"]`,
				) as HTMLElement;
				const arrow = container.querySelector(
					`[data-combobox-arrow="${comboboxId}"]`,
				) as HTMLElement;

				if (
					!displayInput ||
					!valueInput ||
					!dropdown ||
					!searchInput ||
					!optionsList
				)
					return;

				let isOpen = false;
				const allOptions: {
					value: string;
					label: string;
					element: HTMLElement;
				}[] = [];

				// Collect all options
				const optionElements = optionsList.querySelectorAll("li[data-value]");
				for (const li of optionElements) {
					const value = li.getAttribute("data-value") || "";
					const label = li.getAttribute("data-label") || "";
					allOptions.push({ value, label, element: li as HTMLElement });
				}

				// Set initial display value if there's a selected value
				if (valueInput.value) {
					const selectedOption = allOptions.find(
						(opt) => opt.value === valueInput.value,
					);
					if (selectedOption) {
						displayInput.value = selectedOption.label;
					}
				}

				// Toggle dropdown
				const toggleDropdown = () => {
					isOpen = !isOpen;
					dropdown.classList.toggle("hidden", !isOpen);
					if (arrow) {
						arrow.style.transform = isOpen ? "rotate(180deg)" : "rotate(0deg)";
					}

					if (isOpen) {
						searchInput.focus();
						searchInput.value = "";
						filterOptions("");
					}
				};

				// Filter options based on search
				const filterOptions = (searchTerm: string) => {
					const filtered = allOptions.filter((option) =>
						option.label.toLowerCase().includes(searchTerm.toLowerCase()),
					);

					// Hide all options first
					for (const option of allOptions) {
						option.element.style.display = "none";
					}

					// Show filtered options
					for (const option of filtered) {
						option.element.style.display = "block";
					}

					// Show no results message if needed
					const noResultsDiv = dropdown.querySelector(".no-results-message");
					if (filtered.length === 0 && searchTerm) {
						if (!noResultsDiv) {
							const div = document.createElement("div");
							div.className =
								"p-4 text-center text-base-content/60 no-results-message";
							div.textContent = "No results found";
							dropdown.appendChild(div);
						}
					} else if (noResultsDiv) {
						noResultsDiv.remove();
					}
				};

				// Select option
				const selectOption = (option: { value: string; label: string }) => {
					valueInput.value = option.value;
					displayInput.value = option.label;
					isOpen = false;
					dropdown.classList.add("hidden");
					if (arrow) {
						arrow.style.transform = "rotate(0deg)";
					}

					// Trigger change event on the hidden input
					valueInput.dispatchEvent(new Event("change", { bubbles: true }));
				};

				// Event listeners
				displayInput.addEventListener("click", (e) => {
					e.preventDefault();
					toggleDropdown();
				});

				displayInput.addEventListener("focus", (e) => {
					e.preventDefault();
					if (!isOpen) toggleDropdown();
				});

				searchInput.addEventListener("input", (e) => {
					const target = e.target as HTMLInputElement;
					filterOptions(target.value);
				});

				searchInput.addEventListener("keydown", (e) => {
					if (e.key === "Escape") {
						isOpen = false;
						dropdown.classList.add("hidden");
						if (arrow) {
							arrow.style.transform = "rotate(0deg)";
						}
						displayInput.focus();
					}
				});

				// Add click listeners to options
				for (const option of allOptions) {
					const link = option.element.querySelector("a");
					if (link) {
						link.addEventListener("click", (e) => {
							e.preventDefault();
							selectOption(option);
						});
					}
				}

				// Close dropdown when clicking outside
				document.addEventListener("click", (e) => {
					if (!container.contains(e.target as Node)) {
						isOpen = false;
						dropdown.classList.add("hidden");
						if (arrow) {
							arrow.style.transform = "rotate(0deg)";
						}
					}
				});
			}
		},
	},

	setup: {
		htmx: () => {
			// Setup htmx
		},
		elements: () => {
			wgaInternal.els.dialog = document.getElementById(
				"d",
			) as HTMLDialogElement;
			wgaInternal.els.toastContainer = document.getElementById(
				"toast-container",
			) as HTMLElement;
			wgaInternal.dialogDefaultContent =
				wgaInternal.els.dialog?.innerHTML || "";
		},
	},
};

(() => {
	wgaInternal.func.init();
})();

window.wga = {
	dialog: {
		/**
		 * Opens the dialog.
		 */
		open() {
			// Open the dialog
			setTimeout(() => {
				wgaInternal.els.dialog?.showModal();
			}, 500);
		},
		/**
		 * Closes the dialog and resets its content.
		 */
		close() {
			// Close the dialog
			wgaInternal.els.dialog?.close();
			// Reset the dialog content
			setTimeout(() => {
				if (wgaInternal.els.dialog) {
					wgaInternal.els.dialog.innerHTML = wgaInternal.dialogDefaultContent;
				}
			}, 500);
		},
	},
	window: {
		historyBack() {
			window.history.back();
		},
		close() {
			window.close();
		},
		openPopUp(w: popUpWindow) {
			let newWin = window.open(
				w.url,
				w.title,
				`width=${w.size.width},height=${w.size.height},left=${w.position.x},top=${w.position.y},scrollbars=${w.scrollbars ? "yes" : "no"}, resizable=yes, dependent=yes, toolbar=no, menubar=no, location=no, directories=no, status=no, popup=yes`,
			);

			if (!newWin) {
				return false;
			}

			newWin.opener = this;
			newWin.focus();
			return;
		},
	},
	music: {
		openPopUp() {
			window.wga.window.openPopUp({
				url: "/music",
				title: "Music",
				position: {
					x: 0,
					y: 0,
				},
				size: {
					width: 300,
					height: 400,
				},
			});
		},
	},
};
