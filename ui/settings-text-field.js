import { Controller } from "../web/stimulus.js";
import { notify } from "./note-port.js";
import { matchAddr } from "./live.js";

export default class extends Controller {
	static targets = ["input", "mirror"];
	static values = { url: String, prefix: String, suffix: String };

	#original;
	#canvas;
	#onLiveUpdate;

	connect() {
		this.#original = this.inputTarget.value;
		this.sync();
		this.#onLiveUpdate = (ev) => {
			if (matchAddr(this.element, ev.detail.addr)) {
				this.#serverUpdated(ev.detail.text);
			}
		};
		// Manual listener avoids repeating the action attr on every instance.
		document.addEventListener("live:update", this.#onLiveUpdate);
	}

	disconnect() {
		document.removeEventListener("live:update", this.#onLiveUpdate);
	}

	#serverUpdated(value) {
		this.#original = value;
		const input = this.inputTarget;
		if (input === document.activeElement) return;
		input.value = value;
		this.sync();
	}

	save() {
		const input = this.inputTarget;
		const value = input.value.trim();
		if (value === this.#original) {
			this.dispatch("cancel");
			return;
		}
		if (value === "") {
			input.value = this.#original;
			this.sync();
			this.dispatch("cancel");
			return;
		}

		// Optimistic: accept the new value immediately.
		const was = this.#original;
		this.#original = value;
		input.value = value;
		this.sync();

		const data = new FormData();
		for (const el of this.element.querySelectorAll("input[type=hidden]")) {
			data.set(el.name, el.value);
		}
		data.set(input.name, value);
		input.disabled = true;
		fetch(this.urlValue, { method: "POST", body: data }).then(
			(resp) => {
				if (!resp.ok) {
					this.#original = was;
					input.value = was;
					this.sync();
					notify("Something went wrong");
					this.dispatch("error");
				} else {
					this.dispatch("commit");
				}
				input.disabled = false;
			},
			() => {
				this.#original = was;
				input.value = was;
				this.sync();
				input.disabled = false;
				notify("Could not reach the server");
				this.dispatch("error");
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
