import * as bulmaToast from "bulma-toast";

const wga = {
  els: {},
  existingCloners: [],
  dialogDefaultContent: "",
};

bulmaToast.setDefaults({
  duration: 5000,
  position: "top-right",
  closeOnClick: true,
  dismissible: true,
  animate: { in: "fadeIn", out: "fadeOut" },
});

wga.els.dialog = document.getElementById("d");
wga.els.dialogDefaultContent = wga.els.dialog.innerHTML;

htmx.config.globalViewTransitions = true;
htmx.config.selfRequestsOnly = true;
htmx.config.allowScriptTags = false;

initNavbar();
initJumpToTop();
InitEventListeners();

/**
 * Initializes the navbar burger elements and adds a click event listener to each of them.
 * @function
 * @returns {void}
 */
function initNavbar() {
  // Get all "navbar-burger" elements
  const $navbarBurgers = document.querySelectorAll(".navbar-burger");

  /**
   * Toggles the "is-active" class on the target element and its corresponding button.
   * @function
   * @param {Event} event - The event object.
   * @returns {void}
   */
  function classToggler(event) {
    //find nearest `a` parent of event target
    const target = event.target.closest("a") || event.target;

    // Get the target from the "data-target" attribute
    const dataTarget = target.dataset.target;
    const $target = document.getElementById(dataTarget);

    // Toggle the "is-active" class on both the "navbar-burger" and the "navbar-menu"
    target.classList.toggle("is-active");
    $target.classList.toggle("is-active");
  }

  // Add a click event on each of them
  $navbarBurgers.forEach((el) => {
    el.removeEventListener("click", classToggler);
    el.addEventListener("click", classToggler);
  });
}

/**
 * Initializes the Viewer plugin on all elements with the `data-viewer` attribute.
 * @function
 * @returns {void}
 */
function initViewer() {
  const elements = document.querySelectorAll("[data-viewer]");
  if (elements.length > 0) {
    elements.forEach((element) => {
      new Viewer(element, {
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

/**
 * Initializes event listeners for the postcard dialog success and error events, as well as the htmx:load event and the trix-before-initialize event.
 * @function
 * @name InitEventListeners
 * @returns {void}
 */
function InitEventListeners() {
  document.body.addEventListener("notification:toast", function (evt) {
    if (evt.detail.closeDialog) {
      wga.els.dialog.close();
    }

    bulmaToast.toast({
      message: evt.detail.message,
      type: evt.detail.type,
    });
  });

  document.body.addEventListener("htmx:load", function (evt) {
    initViewer();
    initCloner();
  });

  document.addEventListener("trix-before-initialize", () => {
    Trix.config.toolbar.getDefaultHTML = () => {
      return `
    <div class="trix-button-row">
      <span class="trix-button-group trix-button-group--text-tools" data-trix-button-group="text-tools">
        <button type="button" class="trix-button trix-button--icon trix-button--icon-bold" data-trix-attribute="bold" data-trix-key="b" title="Bold" tabindex="-1">Bold</button>
        <button type="button" class="trix-button trix-button--icon trix-button--icon-italic" data-trix-attribute="italic" data-trix-key="i" title="Italic" tabindex="-1">Italic</button>
        <button type="button" class="trix-button trix-button--icon trix-button--icon-strike" data-trix-attribute="strike" title="Strike" tabindex="-1">Strike</button>
      </span>

      <span class="trix-button-group trix-button-group--block-tools" data-trix-button-group="block-tools">
        <button type="button" class="trix-button trix-button--icon trix-button--icon-heading-1" data-trix-attribute="heading1" title="Heading 1" tabindex="-1">Heading 1</button>
        <button type="button" class="trix-button trix-button--icon trix-button--icon-quote" data-trix-attribute="quote" title="Quote" tabindex="-1">Quote</button>
      </span>

    </div>`;
    }; // Change Trix.config if you need
  });

  document.addEventListener("DOMContentLoaded", (event) => {
    const urlParams = new URLSearchParams(window.location.search);
    const checkbox = document.getElementById("dualModeCheckbox");

    for (let param of urlParams.entries()) {
      if (param[0] === "left" && param[1] === "right") {
        checkbox.checked = true;
        break;
      }
    }
  });
}

function initCloner() {
  // find all the elements with data-cloner-target attribute
  const cloners = document.querySelectorAll("[data-cloner-target]");

  // remove not seen items from wga.existingCloners
  wga.existingCloners = wga.existingCloners.filter((el) =>
    document.body.contains(el),
  );

  cloners.forEach((cloner) => {
    // get the target element
    const target = document.querySelector(cloner.dataset.clonerTarget);

    // get the target's innetHTML as the template
    const template = target.innerHTML;

    // if target not in wga.existingCloners

    if (!wga.existingCloners.includes(cloner.dataset.clonerTarget)) {
      // add target to wga.existingCloners
      wga.existingCloners.push(target);

      cloner.addEventListener("click", () => {
        // append the template to the target
        target.insertAdjacentHTML("beforeend", template);

        // find all the elements with data-cloner-remove-me attribute
        const removeMe = target.querySelectorAll("[data-cloner-remove-me]");

        // loop through each removeMe

        removeMe.forEach((el) => {
          const removeMe = () => {
            // find the closest .field element
            const field = el.closest(".field");

            // remove the field
            field.remove();
          };

          el.removeEventListener("click", removeMe);

          // add click event listener
          el.addEventListener("click", removeMe);
        });
      });
    }
  });
}

window.wga = {
  openDialog() {
    wga.els.dialog.showModal();
  },
  closeDialog() {
    wga.els.dialog.close();
    wga.els.dialog.innerHTML = wga.els.dialogDefaultContent;
  },
  windowHistoryBack() {
    window.history.back();
  },
  windowClose() {
    window.close();
  },
  dualMode: {
    changeLinkTargets(checkbox) {
      let currentUrl = new URL(window.location.href);
      let params = currentUrl.searchParams;

      let newParams = new URLSearchParams(params.toString()); // create a copy of current params

      if (checkbox.checked) {
        newParams.append("left", "right"); // append new 'left' param
      } else {
        // remove the 'left=right' param if it exists
        let paramsArray = [...newParams.entries()];
        paramsArray = paramsArray.filter(
          (param) => !(param[0] === "left" && param[1] === "right"),
        );
        newParams = new URLSearchParams(paramsArray);
      }

      currentUrl.search = newParams.toString();
      const newUrl = currentUrl.toString();

      // get tag with id lc-area
      const alma = htmx.ajax("GET", newUrl, checkbox);
      return alma;
      // leftContent.innerHTML = alma;

      // checkbox.setAttribute('hx-get', newUrl);
      // htmx.trigger(checkbox, 'htmx:trigger');

      // xhr.addEventListener('load', function() {
      //   var response = xhr.response;
      //   var leftContent = document.querySelector('.LeftContent');
      //   leftContent.innerHTML = response;
      // });

      // var xhr = htmx.ajax('GET', newUrl, checkbox);
      // xhr.onload = function() {
      //   if (xhr.status >= 200 && xhr.status < 400) {
      //     // The request has been completed successfully
      //     console.log(xhr);
      //     console.log(xhr.responseText);
      //     // var leftContent = document.querySelector('.LeftContent');
      //     // leftContent.innerHTML = response;
      //   } else {
      //     // There was an error with the request
      //     console.error('Server responded with status: ' + xhr.status);
      //   }
      // }
      // xhr.onerror = function() {
      //   // There was a connection error
      //   console.error('There was a connection error');
      // }
    },
    changeLinkTargets2(checkbox) {
      let currentUrl = new URL(window.location.href);
      let params = currentUrl.searchParams;

      let newParams = new URLSearchParams(params.toString()); // create a copy of current params

      if (checkbox.checked) {
        newParams.append("left", "right"); // append new 'left' param
      } else {
        // remove the 'left=right' param if it exists
        let paramsArray = [...newParams.entries()];
        paramsArray = paramsArray.filter(
          (param) => !(param[0] === "left" && param[1] === "right"),
        );
        newParams = new URLSearchParams(paramsArray);
      }

      currentUrl.search = newParams.toString();
      return currentUrl.toString();
    },
  },
  music: {
    openMusicWindow() {
      let w = window.open(
        `/musics`,
        `newWin`,
        `scrollbars=yes,status=no,dependent=no,screenX=0,screenY=0,width=420,height=300`,
      );
      w.opener = this;
      w.focus();
      return false;
    },
  },
};
