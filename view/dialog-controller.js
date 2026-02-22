import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	dismiss() {
		this.element.classList.add("hidden");
	}
}
