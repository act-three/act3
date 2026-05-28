import { Controller } from "../web/stimulus.js";
import { active, start } from "./video-upload.js";

export default class extends Controller {
	static targets = ["picker", "button"];

	connect() {
		const state = active();
		if (state) this.#apply(state.target);
	}

	open() {
		this.pickerTarget.click();
	}

	upload(event) {
		const isVideo = !!this.element.querySelector("input[name='video']");
		if (!isVideo) {
			this.element.requestSubmit(this.buttonTarget);
			return;
		}
		event?.preventDefault?.();
		start(this.element);
		this.element.reset();
	}

	reset() {
		this.element.reset();
	}

	// onUploadStart and onUploadEnd are wired up by video upload
	// forms (see uploadVideoForm) to keep the picker button in sync
	// with the global upload state. Torrent and image forms have no
	// data-upload-target and don't subscribe.
	onUploadStart({ detail }) {
		this.#apply(detail.target);
	}

	onUploadEnd() {
		this.buttonTarget.hidden = false;
		this.buttonTarget.disabled = false;
	}

	#apply(activeTarget) {
		const mine = this.element.dataset.uploadTarget;
		if (!mine) return;
		if (mine === activeTarget) {
			this.buttonTarget.hidden = true;
		} else {
			this.buttonTarget.disabled = true;
		}
	}
}
