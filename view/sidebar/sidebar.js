import { Controller } from "../../web/stimulus.js";

export default class extends Controller {
	static targets = ["link"];

	initialize() {
		this.#showSelected(document.location);
	}

	visit(ev) {
		this.#showSelected(new URL(ev.detail.url));
	}

	#showSelected(url) {
		const current = this.#containingPaths(url.pathname);
		for (const t of this.linkTargets) {
			const path = t.getAttribute("href");
			if (current.has(path)) {
				t.setAttribute("data-selected", "");
			} else {
				t.removeAttribute("data-selected");
			}
		}
	}

	#containingPaths(path) {
		let s = new Set();
		while (path != "") {
			s.add(path);
			path = this.#dirname(path);
		}
		return s
	}

	#dirname(path) {
		const n = path.lastIndexOf("/");
		if (n < 0) {
			return "";
		}
		return path.substring(0, n);
	}
}
