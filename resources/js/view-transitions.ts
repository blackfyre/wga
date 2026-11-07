// HTMX ignores the returned transition. A skipped capture still runs its DOM
// update, but ready rejects and would otherwise become an unhandled rejection.
export function guardViewTransitions(): void {
  const start = document.startViewTransition;
  if (!start) return;
  document.startViewTransition = function (...args) {
    const transition = start.apply(this, args);
    void transition.ready.catch(async (error: unknown) => {
      // Callback failures must remain observable, even InvalidStateError.
      await transition.updateCallbackDone;
      if (!(
        error instanceof DOMException && error.name === "InvalidStateError"
      )) {
        throw error;
      }
    });
    return transition;
  };
}
