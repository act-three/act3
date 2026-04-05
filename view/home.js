import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static values = { mode: String };
	static targets = ["movie", "series", "noResults"];

	#searching = false;
	#anyMovie = false;
	#anySeries = false;
	#anyCollection = false;

	connect() {
		for (const el of this.element.querySelectorAll(".v-poster-grid-poster")) {
			if (el.dataset.kind === "movie") this.#anyMovie = true;
			if (el.dataset.kind === "series") this.#anySeries = true;
		}
	}

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
		this.updateNoResults();
	}

	search(e) {
		const terms = e.target.value.toLowerCase().split(/\s+/).filter(Boolean);
		this.#searching = terms.length > 0;
		this.#anyMovie = false;
		this.#anySeries = false;
		this.#anyCollection = false;
		for (const el of this.element.querySelectorAll(".v-poster-grid-poster")) {
			const words = (el.dataset.title || "").toLowerCase().split(/\s+/);
			const match = terms.every(t => words.some(w => w.startsWith(t)));
			const hidden = this.#searching && !match;
			el.toggleAttribute("data-search-hidden", hidden);
			if (!hidden && el.dataset.kind === "movie") this.#anyMovie = true;
			if (!hidden && el.dataset.kind === "series") this.#anySeries = true;
		}
		for (const el of this.element.querySelectorAll(".v-collection-banner-x")) {
			const words = (el.dataset.title || "").toLowerCase().split(/\s+/);
			const match = terms.every(t => words.some(w => w.startsWith(t)));
			const hidden = !this.#searching || !match;
			el.toggleAttribute("data-search-hidden", hidden);
			if (!hidden) this.#anyCollection = true;
		}
		this.updateNoResults();
	}

	updateNoResults() {
		if (!this.#searching) {
			this.noResultsTarget.toggleAttribute("data-visible", false);
			return;
		}
		const mode = this.modeValue;
		let any = !mode && this.#anyCollection;
		if (!mode || mode === "movie") any = any || this.#anyMovie;
		if (!mode || mode === "series") any = any || this.#anySeries;
		this.noResultsTarget.toggleAttribute("data-visible", !any);
	}
}
