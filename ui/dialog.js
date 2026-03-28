import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	connect() {
		this.element.show();
		this.#onKeydown = (e) => {
			if (e.key === "Escape") this.close();
		};
		document.addEventListener("keydown", this.#onKeydown);
	}

	disconnect() {
		document.removeEventListener("keydown", this.#onKeydown);
	}

	close() {
		this.element.close();
		this.element.remove();
	}

	backdropClose(event) {
		if (event.target === this.element) {
			this.close();
		}
	}

	#onKeydown;
}
