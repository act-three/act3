import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	connect() {
		this.element.show();
	}

	close() {
		this.element.close();
		this.element.remove();
	}
}
