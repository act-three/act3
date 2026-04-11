import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	#pending = false;

	attach() {
		const track = this.element.querySelector("[role=switch]");
		if (!track || track.getAttribute("aria-checked") === "true") {
			return;
		}
		this.#pending = true;
		track.click();
	}

	commit() {
		if (!this.#pending) return;
		this.#pending = false;
		const dialog = this.element.closest("dialog");
		if (dialog) {
			dialog.close();
			dialog.remove();
		}
	}
}
