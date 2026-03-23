import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static targets = ["item"];
	static values = {
		prefix: String,
		target: String,
		selected: String,
	};

	initialize() {
		this.#initSelected();
		this.#showSelected();
	}

	selectedValueChanged() {
		this.#showSelected();
	}

	select(ev) {
		this.selectedValue = ev.params.id;
		this.#navigate();
	}

	render() {
		this.#initSelected();
	}

	#initSelected() {
		const path = document.location.pathname;
		if (!path.startsWith(this.prefixValue)) {
			this.selectedValue = "";
			return;
		}
		for (const t of this.itemTargets) {
			const url = t.getAttribute("data-list-url-param");
			if (path === url || path.startsWith(url + "/")) {
				this.selectedValue = t.getAttribute("data-list-id-param");
				return;
			}
		}
		this.selectedValue = "";
	}

	#navigate() {
		if (this.selectedValue === "") return;
		for (const t of this.itemTargets) {
			if (t.getAttribute("data-list-id-param") === this.selectedValue) {
				const url = t.getAttribute("data-list-url-param");
				Turbo.visit(url, { frame: this.targetValue });
				return;
			}
		}
	}

	#showSelected() {
		for (const t of this.itemTargets) {
			const id = t.getAttribute("data-list-id-param");
			if (id === this.selectedValue) {
				t.setAttribute("data-selected", "");
			} else {
				t.removeAttribute("data-selected");
			}
		}
	}
}
