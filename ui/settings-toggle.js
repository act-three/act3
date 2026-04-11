import { Controller } from "../web/stimulus.js";
import { notify } from "./note-port.js";

export default class extends Controller {
	static targets = ["track", "input"];
	static values = { url: String };

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

		const animated = new Promise((resolve) => {
			track.addEventListener("transitionend", resolve, { once: true });
			setTimeout(resolve, 200);
		});

		const data = new FormData(this.element);
		fetch(this.urlValue, { method: "POST", body: data }).then(
			async (resp) => {
				if (!resp.ok) {
					track.setAttribute("aria-checked", String(was));
					input.value = String(was);
					notify("Something went wrong");
				} else {
					await animated;
					this.dispatch("commit");
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
