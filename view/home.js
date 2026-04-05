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
		const q = e.target.value.toLowerCase();
		for (const el of this.element.querySelectorAll(".v-poster-grid-poster")) {
			const title = (el.dataset.title || "").toLowerCase();
			el.toggleAttribute("data-search-hidden", q !== "" && !title.includes(q));
		}
		for (const el of this.element.querySelectorAll(".v-collection-banner-x")) {
			const title = (el.dataset.title || "").toLowerCase();
			el.toggleAttribute("data-search-hidden", q === "" || !title.includes(q));
		}
	}
}
