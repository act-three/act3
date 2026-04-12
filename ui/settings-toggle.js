import { Controller } from "../web/stimulus.js";
import { notify } from "./note-port.js";

export default class extends Controller {
	static targets = ["track"];
	static values = { url: String, name: String, params: Object };

	toggle() {
		const track = this.trackTarget;
		if (track.disabled) return;

		const was = track.getAttribute("aria-checked") === "true";
		const now = !was;

		// Optimistic flip.
		track.setAttribute("aria-checked", String(now));
		track.disabled = true;

		const animated = new Promise((resolve) => {
			track.addEventListener("transitionend", resolve, { once: true });
			setTimeout(resolve, 200);
		});

		const body = new URLSearchParams({
			...this.paramsValue,
			[this.nameValue]: String(now),
		});
		fetch(this.urlValue, { method: "POST", body }).then(
			async (resp) => {
				if (!resp.ok) {
					track.setAttribute("aria-checked", String(was));
					notify("Something went wrong");
				} else {
					await animated;
					this.dispatch("commit");
				}
				track.disabled = false;
			},
			() => {
				track.setAttribute("aria-checked", String(was));
				track.disabled = false;
				notify("Could not reach the server");
			},
		);
	}
}
