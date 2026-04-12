import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static values = { url: String };

	async open() {
		try {
			const resp = await fetch(this.urlValue, {
				headers: { Accept: "text/vnd.turbo-stream.html" },
			});
			if (resp.ok) {
				Turbo.renderStreamMessage(await resp.text());
			}
		} catch {
			// Network error — the dialog simply doesn't open.
		}
	}
}
