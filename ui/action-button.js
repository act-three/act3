import { Controller } from "../web/stimulus.js";
import { notify } from "./note-port.js";

export default class extends Controller {
	static values = { url: String, params: Object };

	async call() {
		this.element.disabled = true;
		const body = new URLSearchParams(this.paramsValue);
		try {
			const resp = await fetch(this.urlValue, {
				method: "POST",
				body,
				headers: { Accept: "text/vnd.turbo-stream.html" },
			});
			if (resp.redirected) {
				Turbo.visit(resp.url);
				return;
			}
			if (!resp.ok) {
				notify("Something went wrong");
				this.element.disabled = false;
				return;
			}
			const ct = resp.headers.get("Content-Type") || "";
			if (ct.includes("turbo-stream")) {
				Turbo.renderStreamMessage(await resp.text());
			}
		} catch {
			notify("Could not reach the server");
		}
		this.element.disabled = false;
	}
}
