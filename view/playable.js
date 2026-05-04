import { Controller } from "../web/stimulus.js";

// playable wires the pre-player audio + subtitle Selects on the
// theater page to the play button's href. select:change events from
// either Select trigger #updateHref, which rebuilds the href as
// baseUrl?a=<audioRenditionID>&s=<subtitleID>. On connect we also
// stamp in pin_audio=1 when the browser lacks the audioTracks API
// (Chrome): the player view uses that to bake the audio rendition
// into the SSR <source> URL so Chrome's first manifest fetch is
// already pinned. Safari, which does have the API, gets the unpinned
// manifest so it can switch audio without a source-swap.
export default class extends Controller {
	static targets = ["playButton", "audioSelect", "subtitleSelect"];
	static values = { baseUrl: String };

	connect() {
		this.updateHref();
	}

	updateHref() {
		if (!this.hasPlayButtonTarget) return;
		const url = new URL(this.baseUrlValue, location.href);
		if (this.hasAudioSelectTarget) {
			url.searchParams.set("a", this.#selectValue(this.audioSelectTarget));
		}
		if (this.hasSubtitleSelectTarget) {
			url.searchParams.set("s", this.#selectValue(this.subtitleSelectTarget));
		}
		if (!this.#hasAudioTracksAPI) {
			url.searchParams.set("pin_audio", "1");
		}
		this.playButtonTarget.href = url.toString();
	}

	#selectValue(el) {
		// Mirrors the Select controller's currentValue without needing
		// a cross-controller outlet — the value lives on the data attr.
		return el.dataset.selectCurrentValue ?? "";
	}

	get #hasAudioTracksAPI() {
		return "audioTracks" in document.createElement("video");
	}
}
