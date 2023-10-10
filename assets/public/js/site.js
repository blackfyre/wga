htmx.onLoad(function (content) {
    initNavbar();
    initViewer();
    initJumpToTop();
    removeNotification();
});

htmx.config.globalViewTransitions = true;

function initNavbar () {
    // Get all "navbar-burger" elements
    const $navbarBurgers = Array.prototype.slice.call(document.querySelectorAll('.navbar-burger'), 0);

    // Add a click event on each of them
    $navbarBurgers.forEach(el => {
        el.addEventListener('click', () => {

            // Get the target from the "data-target" attribute
            const target = el.dataset.target;
            const $target = document.getElementById(target);

            // Toggle the "is-active" class on both the "navbar-burger" and the "navbar-menu"
            el.classList.toggle('is-active');
            $target.classList.toggle('is-active');

        });
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