import { Controller } from "../web/stimulus.js";
import { active } from "./video-upload.js";

// upload-progress mirrors the in-flight upload's progress.
// An element's data-upload-target binds it to a specific upload
// target; an empty / missing attribute means show progress for
// any upload (used by the sidebar and theater nav).

export default class extends Controller {
	static targets = ["fill", "label"];

	connect() {
		const state = active();
		if (state && this.#matches(state.target)) {
			this.#show(state.progress, state.filename);
		}
	}

	start({ detail }) {
		if (!this.#matches(detail.target)) return;
		this.#show(detail.progress, detail.filename);
	}

	progress({ detail }) {
		if (!this.#matches(detail.target)) return;
		this.#updateBar(detail.progress);
	}

	end() {
		this.element.hidden = true;
	}

	#matches(target) {
		const mine = this.element.dataset.uploadTarget;
		return !mine || mine === target;
	}

	#show(progress, filename) {
		this.element.hidden = false;
		if (this.hasLabelTarget) {
			this.labelTarget.textContent = filename;
		}
		this.#updateBar(progress);
	}

	#updateBar(frac) {
		const pct = Math.max(0, Math.min(1, frac)) * 100;
		this.fillTarget.style.width = `${pct.toFixed(1)}%`;
	}
}
