import { Controller } from "../web/stimulus.js";
import { notify } from "./note-port.js";

export default class extends Controller {
	static targets = ["input", "mirror"];
	static values = { url: String, prefix: String, suffix: String, text: String };

	#original;
	#canvas;

	connect() {
		this.#original = this.inputTarget.value;
		this.sync();
	}

	textValueChanged(value) {
		if (!this.hasTextValue) return;
		this.#original = value;
		const input = this.inputTarget;
		if (input === document.activeElement) return;
		input.value = value;
		this.sync();
	}

	save() {
		const input = this.inputTarget;
		const value = input.value.trim();
		if (value === this.#original) return;
		if (value === "") {
			input.value = this.#original;
			this.sync();
			return;
		}

		// Optimistic: accept the new value immediately.
		const was = this.#original;
		this.#original = value;
		input.value = value;
		this.sync();

		const data = new FormData(this.element);
		input.disabled = true;
		input.dataset.optimistic = "";
		setTimeout(() => delete input.dataset.optimistic, 150);
		fetch(this.urlValue, { method: "POST", body: data }).then(
			(resp) => {
				if (!resp.ok) {
					this.#original = was;
					input.value = was;
					this.sync();
					notify("Something went wrong");
				}
				input.disabled = false;
			},
			() => {
				this.#original = was;
				input.value = was;
				this.sync();
				input.disabled = false;
				notify("Could not reach the server");
			},
		);
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
			this.inputTarget.value = this.#original;
			this.inputTarget.blur();
		}
	}
}
