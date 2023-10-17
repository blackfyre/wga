
import * as bulmaToast from 'bulma-toast'

const wga = {
    els: {}
};

bulmaToast.setDefaults({
    duration: 2000,
    position: 'top-left',
    closeOnClick: true,
})

document.body.addEventListener('htmx:load', function (evt) {

    wga.els.dialog = document.getElementById("d");
    initNavbar();
    initViewer();
    initJumpToTop();
    InitEventListeners();
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

    </div>`
    }  // Change Trix.config if you need
})

htmx.config.globalViewTransitions = true;
htmx.config.selfRequestsOnly = true;
htmx.config.allowScriptTags = false;

function classToggler (event) {

    //find nearest `a` parent of event target
    const target = event.target.closest('a') || event.target;

    // Get the target from the "data-target" attribute
    const dataTarget = target.dataset.target;
    const $target = document.getElementById(dataTarget);

    // Toggle the "is-active" class on both the "navbar-burger" and the "navbar-menu"
    target.classList.toggle('is-active');
    $target.classList.toggle('is-active');

};

function initNavbar () {
    // Get all "navbar-burger" elements
    const $navbarBurgers = document.querySelectorAll('.navbar-burger');

    // Add a click event on each of them
    $navbarBurgers.forEach(el => {
        el.removeEventListener('click', classToggler);
        el.addEventListener("click", classToggler);
    });
}

function initViewer () {
    const elements = document.querySelectorAll('[data-viewer]');
    if (elements.length > 0) {
        elements.forEach(element => {
            new Viewer(element, {
                toolbar: {
                    zoomIn: 1,
                    zoomOut: 1,
                    oneToOne: 1,
                    reset: 1,
                    prev: 1,
                    play: {
                        show: 1,
                        size: 'large',
                    },
                    next: 1,
                    rotateLeft: 1,
                    rotateRight: 1,
                    flipHorizontal: 0,
                    flipVertical: 0,
                },
            })
        })
    }

}

function initJumpToTop () {
    const jumpToTop = document.querySelector('.jump.back-to-top');
    if (jumpToTop) {
        jumpToTop.addEventListener('click', () => {
            window.scrollTo({ top: 0, behavior: 'smooth' });
        })
    }
}

function removeNotification () {
    document.addEventListener('DOMContentLoaded', () => {
        (document.querySelectorAll('.notification .delete') || []).forEach(($delete) => {
            const $notification = $delete.parentNode;

            $delete.addEventListener('click', () => {
                $notification.parentNode.removeChild($notification);
            });
        });
    });
}

function DualModeListeners () {
    const dualModeToggles = document.getElementsByClassName('toggle-dual');
    const body = document.querySelector('body');
    const dualModeSection = document.getElementById('dual');

    if (dualModeToggles.length > 0) {
        for (let i = 0; i < dualModeToggles.length; i++) {

            //remove click event from all dual mode toggles
            dualModeToggles[i].removeEventListener('click', (e) => { });

            dualModeToggles[i].addEventListener('click', (e) => {
                e.preventDefault();

            })
        }
    }
}

function ToggleDualMode () {
    const html = document.querySelector('html');
    const body = document.querySelector('body');
    const dualModeSection = document.getElementById('dual');

    if (body.classList.contains('has-active-backdrop')) {
        html.classList.remove('is-clipped');
        body.classList.remove('has-active-backdrop');
        dualModeSection.classList.remove('is-active');
    } else {
        html.classList.add('is-clipped');
        body.classList.add('has-active-backdrop');
        dualModeSection.classList.add('is-active');
    }
}

function InitEventListeners () {
    document.body.addEventListener("postcard:dialog:success", function (evt) {
        wga.els.dialog.close();

        bulmaToast.toast({
            message: evt.details.message,
            type: 'is-success',
            dismissible: true,
            animate: { in: 'fadeIn', out: 'fadeOut' },
        })
    })
}