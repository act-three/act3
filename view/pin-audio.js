import { Controller } from "../web/stimulus.js";

// pin-audio sets the hidden pin_audio field when the browser can't
// switch audio tracks client-side (no audioTracks API), so the audio
// rendition has to be pinned in the manifest instead.
export default class extends Controller {
	static targets = ["pinAudio"];

	connect() {
		if (!this.hasPinAudioTarget) return;
		if (!("audioTracks" in document.createElement("video"))) {
			this.pinAudioTarget.value = "1";
		}
	}
}
