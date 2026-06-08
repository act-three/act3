import { Controller } from "../web/stimulus.js";

// Presentation-only companion to the settings text field. Saving rides
// the input's change event through domi; this controller handles the
// prefix/suffix mirror overlay and keyboard ergonomics. The value
// attribute always holds the last server-rendered value, so Escape
// reverts to it.
export default class extends Controller {
	static targets = ["input", "mirror"];
	static values = { prefix: String, suffix: String };

	#canvas;

	connect() {
		this.sync();
	}

	sync() {
		if (!this.hasMirrorTarget) return;
		const prefix = this.prefixValue;
		const suffix = this.suffixValue;
		const v = this.inputTarget.value;

		if (prefix) {
			const pw = this.#measureText(prefix);
			this.inputTarget.style.textIndent = pw + "px";
			this.mirrorTarget.value = prefix;
		} else if (suffix) {
			this.mirrorTarget.value = v ? suffix : "";
			this.mirrorTarget.style.textIndent = v
				? this.#measureText(v) + "px"
				: "0";
		}
	}

	#measureText(text) {
		if (!this.#canvas) {
			this.#canvas = document.createElement("canvas");
		}
		const ctx = this.#canvas.getContext("2d");
		ctx.font = getComputedStyle(this.inputTarget).font;
		return ctx.measureText(text).width;
	}

	keydown(ev) {
		if (ev.key === "Enter") {
			ev.preventDefault();
			this.inputTarget.blur();
		} else if (ev.key === "Escape") {
			this.inputTarget.value = this.inputTarget.getAttribute("value") ?? "";
			this.sync();
			this.inputTarget.blur();
		}
	}
}
