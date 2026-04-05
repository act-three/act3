import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static values = { mode: String };
	static targets = ["regular", "special"];

	setRegular() {
		this.modeValue = this.modeValue === "regular" ? "" : "regular";
	}
	setSpecial() {
		this.modeValue = this.modeValue === "special" ? "" : "special";
	}

	modeValueChanged(mode) {
		for (const t of ["regular", "special"]) {
			for (const el of this[`${t}Targets`]) {
				el.toggleAttribute("data-selected", t === mode);
			}
		}
	}
}
