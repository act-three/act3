import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static targets = ["picker", "button"];

	open() {
		this.pickerTarget.click();
	}

	upload() {
		this.element.requestSubmit(this.buttonTarget);
	}

	reset() {
		this.element.reset();
	}
}
