import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	connect() {
		this.element.showModal();
	}

	disconnect() {
		this.element.close();
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
