import { Controller } from "../web/stimulus.js";
import { notify } from "./note-port.js";
import { matchAddr } from "./live.js";

export default class extends Controller {
	static targets = ["button"];
	static values = { url: String, name: String, selected: String };

	#onLiveUpdate;

	connect() {
		this.#syncButtons();
		this.#onLiveUpdate = (ev) => {
			if (matchAddr(this.element, ev.detail.addr)) {
				this.selectedValue = ev.detail.text;
				this.#syncButtons();
			}
		};
		document.addEventListener("live:update", this.#onLiveUpdate);
	}

	disconnect() {
		document.removeEventListener("live:update", this.#onLiveUpdate);
	}

	select(ev) {
		const value = ev.currentTarget.getAttribute("data-value");
		if (value === this.selectedValue) return;

		const was = this.selectedValue;
		this.selectedValue = value;
		this.#syncButtons();
		this.element.setAttribute("disabled", "");
		this.element.dataset.optimistic = "";
		setTimeout(() => delete this.element.dataset.optimistic, 150);

		const data = new FormData();
		// Include hidden inputs from the form.
		for (const input of this.element.querySelectorAll("input[type=hidden]")) {
			data.set(input.name, input.value);
		}
		data.set(this.nameValue, value);

		fetch(this.urlValue, { method: "POST", body: data }).then(
			(resp) => {
				if (!resp.ok) {
					this.selectedValue = was;
					this.#syncButtons();
					notify("Something went wrong");
				}
				this.element.removeAttribute("disabled");
			},
			() => {
				this.selectedValue = was;
				this.#syncButtons();
				this.element.removeAttribute("disabled");
				notify("Could not reach the server");
			},
		);
	}

	#syncButtons() {
		for (const b of this.buttonTargets) {
			if (b.getAttribute("data-value") === this.selectedValue) {
				b.setAttribute("data-selected", "");
			} else {
				b.removeAttribute("data-selected");
			}
		}
	}
}
