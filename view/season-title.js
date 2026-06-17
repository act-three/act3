import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static values = { mode: { type: String, default: "display" } };

	edit() {
		this.modeValue = "edit";
		const input = this.element.querySelector(
			".u-settings-text-field input:not([disabled])",
		);
		input.focus();
		input.select();
	}

	display() {
		this.modeValue = "display";
	}
}
