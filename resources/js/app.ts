import Viewer from "viewerjs";
import Trix from "trix";

declare global {
  interface Window {
    wga: wgaWindow;
  }
}

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
    openPopUp: (w: popUpWindow) => boolean | void;
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
    toast: (message: string, type: string) => void;
    augmentSelects: () => void;
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
    type: string;
  };
}

const deepMerge = (target: object, source: object): object => {
  // Iterate over keys in the source object
  for (const key in source) {
    if (source.hasOwnProperty(key)) {
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

const wgaInternal: wgaInternals = {
  els: {
    dialog: null,
    toastContainer: null,
  },
  existingCloners: [],
  dialogDefaultContent: "",
  eventListeners: [
    () => {
      document.body.addEventListener("notification:toast", function (evt) {
        const event = evt as ToastEvent;
        if (event.detail.closeDialog) {
          wgaInternal.els.dialog?.close();
        }

        wgaInternal.func.toast(event.detail.message, event.detail.type);
      });
    },
    () => {
      document.addEventListener("DOMContentLoaded", function (event) {
        wgaInternal.func.viewer();
        wgaInternal.func.cloner();
        wgaInternal.func.augmentSelects();
      });
    },
    () => {
      document.body.addEventListener("htmx:load", function (evt) {
        wgaInternal.func.viewer();
        wgaInternal.func.cloner();
        wgaInternal.func.augmentSelects();
      });

      document.body.addEventListener("htmx:swapError", function (evt) {
        console.error(evt);
        wgaInternal.func.toast(
          "An error occurred while processing your request.",
          "danger",
        );
      });

      document.body.addEventListener("htmx:targetError", function (evt) {
        console.error(evt);
        wgaInternal.func.toast(
          "An error occurred while processing your request.",
          "danger",
        );
      });

      document.body.addEventListener("htmx:timeout", function (evt) {
        console.error(evt);
        wgaInternal.func.toast("Request timed out!", "danger");
      });
    },
    () => {
      //! This is a workaround, has to be removed when htmx fixes the issue
      addEventListener("htmx:beforeHistorySave", () => {
        document.querySelectorAll(":disabled").forEach((el) => {
          const e = el as HTMLInputElement;
          e.disabled = false;
        });
      });
    },
    () => {
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
    toast: (message: string, type: string) => {
      const toast = document.createElement("div");
      const toastMessage = document.createElement("span");
      toast.className = `alert alert-${type} cursor-pointer`;
      toastMessage.textContent = message;
      toast.appendChild(toastMessage);

      toast.addEventListener("click", () => {
        toast.remove();
      });

      wgaInternal.els.toastContainer?.appendChild(toast);

      setTimeout(() => {
        toast.remove();
      }, 5000);
    },
    init() {
      // Run all internal functions
      wgaInternal.setup.htmx();
      wgaInternal.setup.elements();

      // Run all event listeners
      wgaInternal.eventListeners.forEach((listener) => listener());
    },
    augmentSelects() {
      console.log("Augmenting selects");

      let CreateSelector = (
        rootElement: HTMLElement,
        configObject: wgaComboBoxConfig,
      ) => {
        const defaults: wgaComboBox = {
          input: {
            style:
              "w-full px-4 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500",
            placeholder: "Search...",
            type: "search",
          },
          list: {
            style:
              "absolute z-10 w-full mt-1 bg-white border border-gray-300 rounded shadow max-h-48 overflow-y-auto hidden",
          },
          moreAvailable: {
            style: "px-4 py-2 text-gray-500 hidden",
            content: "...",
          },
          noResults: {
            style: "px-4 py-2 text-gray-500 hidden",
            content: "No results found",
          },
          item: {
            style: "dropdown-item px-4 py-2 cursor-pointer hover:bg-blue-100",
          },
          options: [],
          hooks: {
            onSelected: function (v) {
              console.info(v);
            },
          },
        };

        const c = deepMerge(defaults, configObject) as wgaComboBox;

        let label = rootElement.getAttribute("data-label") || "label";
        let value = rootElement.getAttribute("data-value") || "value";

        if (!Array.isArray(c.options)) {
          throw "Options is not an array";
        }

        if (c.options.length === 0) {
          throw "No options supplied";
        }

        // Create input and attach it to the container element
        const input = document.createElement("input");
        input.className += c.input.style;
        input.setAttribute("placeholder", c.input.placeholder);
        input.setAttribute("type", c.input.type);

        rootElement.appendChild(input);

        // Create the dropdown list element and attach it to the container element
        const dropdownList = document.createElement("div");
        rootElement.appendChild(dropdownList);
        dropdownList.className += c.list.style;

        // Create the list items and attach them to the list
        const dropdownItems: HTMLDivElement[] = [];

        for (var i = 0; i < defaults.options.length; i++) {
          let o = c.options[i];

          let listItem = document.createElement("div");
          listItem.className += c.item.style;
          listItem.setAttribute("data-value", o[value] || "");
          listItem.setAttribute("data-label", o[label] || "");

          listItem.innerHTML = o[label] || "";

          dropdownItems.push(listItem);
          dropdownList.appendChild(listItem);
          //Do something
        }

        // Create the "No results" message and attach it the list
        const noResults = document.createElement("div");
        noResults.innerHTML = c.noResults.content;
        noResults.className += c.noResults.style;

        dropdownList.appendChild(noResults);

        // Create the "More options available" message and attach it the list
        const moreOptions = document.createElement("div");
        moreOptions.className += c.moreAvailable.style;
        moreOptions.innerHTML = c.moreAvailable.content;
        dropdownList.appendChild(moreOptions);

        let selectedIndex = -1;

        // Show dropdown and filter items on input focus or type
        input.addEventListener("focus", showDropdown);
        input.addEventListener("input", showDropdown);

        // Clear search on reset (using native clear button on search type input)
        input.addEventListener("search", showDropdown);

        function showDropdown() {
          const query = input.value.toLowerCase();
          let visibleCount = 0;
          const maxVisibleOptions = 5;

          dropdownItems.forEach((item: HTMLElement) => {
            const text = item.getAttribute("data-label");
            const highlightedText = highlightMatches(text, query);

            if (query === "" || highlightedText !== text) {
              item.innerHTML = highlightedText;
              if (visibleCount < maxVisibleOptions) {
                item.style.display = "block";
                visibleCount++;
              } else {
                item.style.display = "none";
              }
            } else {
              item.style.display = "none";
            }
          });

          // Show or hide "No results" and ellipsis for additional options
          noResults.style.display = visibleCount === 0 ? "block" : "none";
          moreOptions.style.display =
            visibleCount === maxVisibleOptions &&
            dropdownItems.length > maxVisibleOptions
              ? "block"
              : "none";

          dropdownList.style.display = "block";
          selectedIndex = -1;
          setActiveItem(selectedIndex);
        }

        // Function to highlight all matches in a string
        function highlightMatches(text, query) {
          if (!query) return text;

          const regex = new RegExp(`(${query})`, "gi");
          return text.replace(regex, '<span class="bg-yellow-200">$1</span>');
        }

        // Hide dropdown when clicking outside
        document.addEventListener("click", (e: MouseEvent) => {
          if (!e.target) {
            return;
          }

          if (!(e.target as HTMLElement).closest(".relative")) {
            dropdownList.style.display = "none";
          }
        });

        // Select an option
        dropdownItems.forEach((item: HTMLElement) => {
          item.addEventListener("click", () => selectItem(item));
        });

        // Handle keyboard navigation
        input.addEventListener("keydown", function (e: KeyboardEvent) {
          const visibleItems = dropdownItems.filter(
            (item: HTMLElement) => item.style.display !== "none",
          );

          if (e.key === "ArrowDown") {
            e.preventDefault();
            selectedIndex = (selectedIndex + 1) % visibleItems.length;
            setActiveItem(selectedIndex, visibleItems);
          } else if (e.key === "ArrowUp") {
            e.preventDefault();
            selectedIndex =
              (selectedIndex - 1 + visibleItems.length) % visibleItems.length;
            setActiveItem(selectedIndex, visibleItems);
          } else if (e.key === "Enter") {
            e.preventDefault();
            if (selectedIndex > -1) {
              selectItem(visibleItems[selectedIndex]);
            }
          } else if (e.key === "Escape") {
            dropdownList.style.display = "none";
          }
        });

        // Set active item visually
        function setActiveItem(index: number, visibleItems = dropdownItems) {
          dropdownItems.forEach((item: HTMLElement) =>
            item.classList.remove("bg-blue-100"),
          );
          if (index > -1) {
            (visibleItems[index] as HTMLElement).classList.add("bg-blue-100");
          }
        }

        // Select item and close dropdown
        function selectItem(item: HTMLElement) {
          let v = item.getAttribute("data-label");
          input.value = v || "";
          dropdownList.style.display = "none";

          if (
            c.hooks &&
            c.hooks.onSelected &&
            typeof c.hooks.onSelected === "function"
          ) {
            c.hooks.onSelected(item.getAttribute("data-value") || "");
          }
        }
      };

      // Find all the elements with data-choices attribute
      const choices = document.querySelectorAll("[data-combobox]");
      console.log(choices);
      choices.forEach((c) => {
        const selector = c as HTMLDivElement;

        console.log(selector);

        const listId = selector.dataset.choices;

        if (!listId) {
          console.error("data-choices attribute is required");
          return;
        }

        const rawList = document.getElementById(listId);

        if (!rawList) {
          console.error("List not found");
          return;
        }

        // parse the list
        const list = JSON.parse(rawList.innerHTML);

        console.log(list);

        CreateSelector(selector, {
          options: list,
          hooks: {
            onSelected: (v) => {
              console.log(v);
            },
          },
        });
      });
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

(function () {
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
