import htmx from "htmx.org";
import Viewer from "viewerjs";
import Trix from "trix";

declare global {
  interface Window {
    wga: wgaWindow;
  }
}

type wgaWindow = {
  openDialog: () => void;
  closeDialog: () => void;
  windowHistoryBack: () => void;
  windowClose: () => void;
  music: {
    openMusicWindow: () => void;
  };
};

type wgaInternals = {
  els: {
    dialog: HTMLDialogElement | null;
    toastContainer: HTMLElement | null;
  };
  existingCloners: HTMLElement[];
  dialogDefaultContent: string;
};

interface ToastEvent extends Event {
  detail: {
    closeDialog: boolean;
    message: string;
    type: string;
  };
}

const wgaInternal: wgaInternals = {
  els: {
    dialog: null,
    toastContainer: null,
  },
  existingCloners: [],
  dialogDefaultContent: "",
};

(function () {
  wgaInternal.els.dialog = document.getElementById("d") as HTMLDialogElement;
  wgaInternal.els.toastContainer = document.getElementById(
    "toast-container",
  ) as HTMLElement;
  wgaInternal.dialogDefaultContent = wgaInternal.els.dialog?.innerHTML || "";

  htmx.config.globalViewTransitions = true;
  htmx.config.selfRequestsOnly = true;
  htmx.config.allowScriptTags = false;

  InitEventListeners();
})();

/**
 * Initializes the Viewer plugin on all elements with the `data-viewer` attribute.
 * @function
 * @returns {void}
 */
function initViewer() {
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
}

function initCloner() {
  // find all the elements with data-cloner-target attribute
  const cloners = document.querySelectorAll("[data-cloner-target]");

  // remove not seen items from wga.existingCloners
  wgaInternal.existingCloners = wgaInternal.existingCloners.filter((el) =>
    document.body.contains(el),
  );

  cloners.forEach((c) => {
    const cloner = c as HTMLElement;
    // get the target element
    const target = document.querySelector(
      cloner.dataset.clonerTarget as string,
    );

    if (!target) {
      return;
    }

    // get the target's innerHTML as the template
    const template = target.innerHTML;

    // if target not in wga.existingCloners

    if (!wgaInternal.existingCloners.includes(target as HTMLElement)) {
      // add target to wga.existingCloners
      wgaInternal.existingCloners.push(target as HTMLElement);

      cloner.addEventListener("click", () => {
        // append the template to the target
        target?.insertAdjacentHTML("beforeend", template);

        // find all the elements with data-cloner-remove-me attribute
        const removeMe = target.querySelectorAll("[data-cloner-remove-me]");

        // loop through each removeMe

        removeMe.forEach((el) => {
          const removeMe = () => {
            // find the closest .field element
            const field = el.closest("label.input");

            // remove the field
            field?.remove();
          };

          el.removeEventListener("click", removeMe);

          // add click event listener
          el.addEventListener("click", removeMe);
        });
      });
    }
  });
}

/**
 * Initializes event listeners for the postcard dialog success and error events, as well as the htmx:load event and the trix-before-initialize event.
 * @function
 * @name InitEventListeners
 * @returns {void}
 */
function InitEventListeners() {
  document.body.addEventListener("notification:toast", function (evt) {
    const event = evt as ToastEvent;
    if (event.detail.closeDialog) {
      wgaInternal.els.dialog?.close();
    }

    createToast(event.detail.message, event.detail.type);
  });

  document.body.addEventListener("htmx:load", function (evt) {
    initViewer();
    initCloner();
  });

  document.body.addEventListener("htmx:swapError", function (evt) {
    console.error(evt);

    // TODO: show error toast: An error occurred while processing your request.
    createToast("An error occurred while processing your request.", "danger");
  });

  document.body.addEventListener("htmx:targetError", function (evt) {
    console.error(evt);
    // TODO: show error toast: An error occurred while processing your request.
    createToast("An error occurred while processing your request.", "danger");
  });

  document.body.addEventListener("htmx:timeout", function (evt) {
    console.error(evt);
    // TODO: show error toast: Request timed out.
    createToast("Request timed out!", "danger");
  });

  // document.body.addEventListener("htmx:configRequest", (event) => {
  //   const evt = event as CustomEvent;
  //   //get the value of the _csrf cookie
  //   const rawCookies: string = document.cookie;
  //   const cookieList: string[] = rawCookies.split("; ");

  //   if (cookieList.length === 0) {
  //     return;
  //   }

  //   let csrf_token = "";

  //   for (let i = 0; i < cookieList.length; i++) {
  //     const cookie = cookieList[i];
  //     if (cookie.startsWith("_csrf=")) {
  //       csrf_token = cookie.split("=")[1];
  //       break;
  //     }
  //   }

  //   evt.detail.headers["X-XSRF-TOKEN"] = csrf_token;
  // });

  //! This is a workaround, has to be removed when htmx fixes the issue
  addEventListener("htmx:beforeHistorySave", () => {
    document.querySelectorAll(":disabled").forEach((el) => {
      const e = el as HTMLInputElement;
      e.disabled = false;
    });
  });

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
    }; // Change Trix.config if you need
  });
}

/**
 * Initializes the "Jump to Top" functionality.
 * @function
 * @name initJumpToTop
 * @returns {void}
 */
function initJumpToTop() {
  const jumpToTop = document.querySelector(".jump.back-to-top");
  if (jumpToTop) {
    jumpToTop.addEventListener("click", () => {
      window.scrollTo({ top: 0, behavior: "smooth" });
    });
  }
}

function createToast(message: string, type: string) {
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
}

window.wga = {
  openDialog() {
    setTimeout(() => {
      wgaInternal.els.dialog?.showModal();
    }, 500);
  },
  closeDialog() {
    wgaInternal.els.dialog?.close();
    setTimeout(() => {
      if (wgaInternal.els.dialog) {
        wgaInternal.els.dialog.innerHTML = wgaInternal.dialogDefaultContent;
      }
    }, 500);
  },
  windowHistoryBack() {
    window.history.back();
  },
  windowClose() {
    window.close();
  },
  music: {
    openMusicWindow() {
      let w = window.open(
        `/musics`,
        `newWin`,
        `scrollbars=yes,status=no,dependent=no,screenX=0,screenY=0,width=420,height=300`,
      );

      if (!w) {
        return false;
      }

      w.opener = this;
      w.focus();
      return false;
    },
  },
};
