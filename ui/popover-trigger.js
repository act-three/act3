import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static values = { url: String, trigger: String };
	#submitted = false;

	activate({ currentTarget }) {
		this.#submitted = false;
		currentTarget.setAttribute("aria-expanded", "true");
	}

	deactivate({ currentTarget }) {
		if (this.#submitted) return;
		currentTarget.removeAttribute("aria-expanded");
	}

	async open() {
		this.#submitted = true;
		const button = this.element.querySelector(".u-button");
		if (button) button.style.visibility = "visible";

		const params = new URLSearchParams();
		params.set("popover-trigger", this.triggerValue);
		for (const input of this.element.querySelectorAll("input[type=hidden]")) {
			params.set(input.name, input.value);
		}

		try {
			const resp = await fetch(`${this.urlValue}?${params}`, {
				headers: { Accept: "text/vnd.turbo-stream.html" },
			});
			if (resp.ok) {
				document.activeElement?.blur();
				Turbo.renderStreamMessage(await resp.text());
			}
		} catch {
			// Network error — the popover simply doesn't open.
		}
	}
}
