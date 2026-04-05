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
}
