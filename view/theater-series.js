import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static values = { mode: String };
	static targets = ["regular", "special", "all"];

	setRegular() {
		this.modeValue = "regular";
	}
	setSpecial() {
		this.modeValue = "special";
	}
	setAll() {
		this.modeValue = "all";
	}

	modeValueChanged(mode) {
		for (const t of ["regular", "special", "all"]) {
			for (const el of this[`${t}Targets`]) {
				el.toggleAttribute("data-selected", t === mode);
			}
		}
	}
}
