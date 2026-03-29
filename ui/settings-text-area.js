import { Controller } from "../web/stimulus.js";
import { notify } from "./note-port.js";
import { matchAddr } from "./live.js";

export default class extends Controller {
	static targets = ["input"];
	static values = { url: String };

	#original;
	#onLiveUpdate;

	connect() {
		this.#original = this.inputTarget.value;
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
	}

	save() {
		const input = this.inputTarget;
		const value = input.value.trim();
		if (value === this.#original) return;

		// Optimistic: accept the new value immediately.
		const was = this.#original;
		this.#original = value;
		input.value = value;

		const data = new FormData(this.element);
		input.disabled = true;
		input.dataset.optimistic = "";
		setTimeout(() => delete input.dataset.optimistic, 150);
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
		if (ev.key === "Escape") {
			this.inputTarget.value = this.#original;
			this.inputTarget.blur();
		}
	}
}
