import { Controller } from "../web/stimulus.js";
import { notify } from "../ui/note-port.js";

export default class extends Controller {
	static targets = ["picker", "button", "progress", "progressFill"];

	open() {
		this.pickerTarget.click();
	}

	upload(event) {
		// Without a progress target this is the small (image / torrent)
		// upload form: let Turbo handle the submit as before.
		if (!this.hasProgressTarget) {
			this.element.requestSubmit(this.buttonTarget);
			return;
		}
		event?.preventDefault?.();
		this.#uploadXHR();
	}

	reset() {
		this.element.reset();
	}

	#uploadXHR() {
		const form = this.element;
		const xhr = new XMLHttpRequest();
		xhr.open(form.method.toUpperCase(), form.action);
		xhr.setRequestHeader("Accept", "text/vnd.turbo-stream.html");

		const removeNavGuards = this.#installNavGuards();

		this.#setUploading(true);

		xhr.upload.addEventListener("progress", (e) => {
			if (e.lengthComputable) this.#updateProgress(e.loaded / e.total);
		});
		xhr.addEventListener("load", () => {
			if (xhr.status >= 200 && xhr.status < 300) {
				return;
			}
			const ct = xhr.getResponseHeader("Content-Type") || "";
			if (ct.includes("turbo-stream") && xhr.responseText) {
				Turbo.renderStreamMessage(xhr.responseText);
			} else {
				notify("Upload failed");
			}
		});
		xhr.addEventListener("error", () => notify("Could not reach the server"));
		xhr.addEventListener("abort", () => notify("Upload aborted"));
		xhr.addEventListener("loadend", () => {
			removeNavGuards();
			this.#setUploading(false);
			this.reset();
		});
		xhr.send(new FormData(form));
	}

	// Intercepts every way the user can leave the page while the XHR
	// is in flight. Returns a teardown for loadend.
	//
	// Intercepting Turbo's lifecycle events (turbo:before-render etc.)
	// doesn't work cleanly: preventDefault on those *pauses* the
	// render rather than cancelling, leaving the Visit's renderPromise
	// hanging — which leaks the top-of-page progress bar and blocks
	// subsequent navigations on `await view.renderPromise`. So we
	// intercept earlier, at the events that *start* a navigation,
	// where preventDefault means "this click/back never happened" and
	// no Turbo state is ever created.
	//
	//   beforeunload — full document unload (reload, close, off-origin).
	//   The browser shows its own prompt; we just opt in.
	//
	//   click (capture phase) — link clicks and list-item card clicks.
	//   Runs before Turbo's LinkClickObserver / LinkInterceptor and
	//   before Stimulus actions, so cancelling here stops the click
	//   entirely. Filter: <a href> covers explicit links;
	//   [data-list-url-param] covers list-controller cards that
	//   programmatically call Turbo.visit({frame}) on click.
	//
	//   turbo:before-popstate — back/forward. This is an act3 patch
	//   on top of Turbo (see web/turbo.es2017-esm.js): Turbo's native
	//   popstate handling goes through navigator.startVisit, which
	//   skips the cancellable turbo:before-visit event, so there's no
	//   stock hook for restoration visits. Listening on popstate
	//   directly doesn't work either — Turbo's listener is registered
	//   at page load and at-target listeners fire in insertion order
	//   regardless of capture/bubble, so stopImmediatePropagation
	//   runs too late.
	//
	//   On a decline, popstate has already moved the history pointer,
	//   so we restore via history.go(1). That fires another popstate
	//   (and another turbo:before-popstate); a one-shot flag
	//   suppresses the prompt on it.
	#installNavGuards() {
		const msg = "An upload is in progress. Leave anyway?";

		const onBeforeUnload = (e) => {
			e.preventDefault();
			e.returnValue = "";
		};
		window.addEventListener("beforeunload", onBeforeUnload);

		const onClick = (e) => {
			if (!e.target.closest("a[href], [data-list-url-param]")) return;
			if (window.confirm(msg)) return;
			e.preventDefault();
			e.stopImmediatePropagation();
		};
		document.addEventListener("click", onClick, true);

		let suppressNextPopstate = false;
		const onTurboBeforePopstate = (e) => {
			if (suppressNextPopstate) {
				suppressNextPopstate = false;
				e.preventDefault();
				return;
			}
			if (window.confirm(msg)) return;
			e.preventDefault();
			suppressNextPopstate = true;
			history.go(1);
		};
		document.addEventListener("turbo:before-popstate", onTurboBeforePopstate);

		return () => {
			window.removeEventListener("beforeunload", onBeforeUnload);
			document.removeEventListener("click", onClick, true);
			document.removeEventListener("turbo:before-popstate", onTurboBeforePopstate);
		};
	}

	#setUploading(active) {
		this.progressTarget.hidden = !active;
		if (this.hasButtonTarget) this.buttonTarget.hidden = active;
		if (!active) this.#updateProgress(0);
	}

	#updateProgress(frac) {
		const pct = Math.max(0, Math.min(1, frac)) * 100;
		if (this.hasProgressFillTarget) {
			this.progressFillTarget.style.width = `${pct.toFixed(1)}%`;
		}
	}
}
