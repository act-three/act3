import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	connect() {
		this.element.showModal();
		this.element.addEventListener("close", () => this.element.remove(), { once: true });
	}

	close() {
		this.element.close();
	}

	backdropClose(event) {
		if (event.target === this.element) {
			this.close();
		}
	}
}
