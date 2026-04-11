import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	#submitted = false;

	activate({ currentTarget }) {
		this.#submitted = false;
		currentTarget.setAttribute("aria-expanded", "true");
	}

	deactivate({ currentTarget }) {
		if (this.#submitted) return;
		currentTarget.removeAttribute("aria-expanded");
	}

	open() {
		this.#submitted = true;
		const button = this.element.querySelector(".u-button");
		if (!button) return;
		button.style.visibility = "visible";
	}
}
