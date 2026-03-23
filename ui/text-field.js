import { Controller } from "../web/stimulus.js";
import { notify } from "./note-port.js";

export default class extends Controller {
	static targets = ["input"];
	static values = { url: String };

	#original;

	connect() {
		this.#original = this.inputTarget.value;
	}

	save() {
		const input = this.inputTarget;
		const value = input.value.trim();
		if (value === this.#original) return;
		if (value === "") {
			input.value = this.#original;
			return;
		}

		// Optimistic: accept the new value immediately.
		const was = this.#original;
		this.#original = value;
		input.value = value;

		const data = new FormData(this.element);
		input.disabled = true;
		fetch(this.urlValue, { method: "POST", body: data }).then(
			(resp) => {
				if (!resp.ok) {
					this.#original = was;
					input.value = was;
					notify("Something went wrong");
				}
				input.disabled = false;
			},
			() => {
				this.#original = was;
				input.value = was;
				input.disabled = false;
				notify("Could not reach the server");
			},
		);
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
