import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static values = { mode: String };
	static targets = ["overview", "playlist", "frame", "prefetch"];

	setOverview() {
		this.modeValue = "overview";
	}
	setPlaylist() {
		this.modeValue = "playlist";
	}

	prefetch(event) {
		const url = event.currentTarget.dataset.url;
		if (!url) return;
		for (const a of this.prefetchTargets) {
			if (a.href === url || a.getAttribute("href") === url) {
				a.dispatchEvent(new MouseEvent("mouseenter", { bubbles: true }));
				return;
			}
		}
	}

	modeValueChanged(mode) {
		for (const t of ["overview", "playlist"]) {
			for (const el of this[`${t}Targets`]) {
				el.toggleAttribute("data-selected", t === mode);
				if (t === mode) {
					this.frameTarget.src = el.dataset.url;
				}
			}
		}
	}
}
