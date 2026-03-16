import { Controller } from "../web/stimulus.js";
import { notify } from "./note-port.js";

export default class extends Controller {
	static targets = ["track", "input"];

	toggle() {
		const track = this.trackTarget;
		if (track.disabled) return;

		const input = this.inputTarget;
		const was = track.getAttribute("aria-checked") === "true";
		const now = !was;

		// Optimistic flip.
		track.setAttribute("aria-checked", String(now));
		input.value = String(now);
		track.disabled = true;
		track.dataset.animating = "";

		track.addEventListener("transitionend", () => {
			delete track.dataset.animating;
		}, { once: true });

		const data = new FormData(this.element);
		fetch(this.element.action, { method: "POST", body: data }).then(
			(resp) => {
				if (!resp.ok) {
					track.setAttribute("aria-checked", String(was));
					input.value = String(was);
					notify("Something went wrong");
				}
				track.disabled = false;
			},
			() => {
				track.setAttribute("aria-checked", String(was));
				input.value = String(was);
				track.disabled = false;
				notify("Could not reach the server");
			},
		);
	}
}
