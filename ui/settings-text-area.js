import { Controller } from "../web/stimulus.js";

// Presentation-only companion to the settings textarea. Saving rides
// the textarea's change event through domi; this controller only
// handles Escape, reverting to the last server-rendered value (the
// textarea's default value).
export default class extends Controller {
	static targets = ["input"];

	keydown(ev) {
		if (ev.key === "Escape") {
			this.inputTarget.value = this.inputTarget.defaultValue;
			this.inputTarget.blur();
		}
	}
}
