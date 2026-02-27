import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	#ro;

	connect() {
		this.#ro = new ResizeObserver(([e]) => {
			document.body.style.paddingTop = e.contentRect.height + "px";
		});
		this.#ro.observe(this.element);
	}

	disconnect() {
		this.#ro.disconnect();
	}
}
