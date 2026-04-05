import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static values = { mode: String };
	static targets = ["movie", "series"];

	setMovie() {
		this.modeValue = this.modeValue === "movie" ? "" : "movie";
	}
	setSeries() {
		this.modeValue = this.modeValue === "series" ? "" : "series";
	}

	modeValueChanged(mode) {
		for (const t of ["movie", "series"]) {
			for (const el of this[`${t}Targets`]) {
				el.toggleAttribute("data-selected", t === mode);
			}
		}
	}

	search(e) {
		const terms = e.target.value.toLowerCase().split(/\s+/).filter(Boolean);
		for (const el of this.element.querySelectorAll(".v-poster-grid-poster")) {
			const words = (el.dataset.title || "").toLowerCase().split(/\s+/);
			const match = terms.every(t => words.some(w => w.startsWith(t)));
			el.toggleAttribute("data-search-hidden", terms.length > 0 && !match);
		}
		for (const el of this.element.querySelectorAll(".v-collection-banner-x")) {
			const words = (el.dataset.title || "").toLowerCase().split(/\s+/);
			const match = terms.every(t => words.some(w => w.startsWith(t)));
			el.toggleAttribute("data-search-hidden", terms.length === 0 || !match);
		}
	}
}
