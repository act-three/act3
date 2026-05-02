import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static targets = [
		"video",
		"volume",
		"seek",
		"buffer",
		"progress",
		"seekTooltip",
		"currentTime",
		"duration",
		"qualityMenu",
		"captionsMenu",
		"captionsTemplate",
		"audioMenu",
	];
	static values = {
		title: String,
		playing: Boolean,
		paused: Boolean,
		stopped: Boolean,
		harlow: Boolean,
		hideControls: Boolean,
		loading: Boolean,
		currentQuality: String,
		qualityMenuOpen: Boolean,
		currentSubtitle: String,
		captionsMenuOpen: Boolean,
		currentAudio: String,
		audioMenuOpen: Boolean,
	};

	#isTouch = false;
	#timerControls = null;
	#timerLoading = null;
	#harlowMode = false;
	#recentInteraction = false;
	#recentTouchSeek = false;
	#userSeeking = false;
	#lastKey = null;
	#lastSeekTime = Date.now();
	#subtitleTracksDecided = false;

	connect() {
		// When the manifest surfaces a subtitle TextTrack with the
		// same label as one of our captions-template <track> clones,
		// remove the cloned <track> to avoid leaving a duplicate
		// TextTrack. The trackEl.track !== t check distinguishes
		// addtrack events for manifest tracks (where the cloned
		// <track> is bound to a *different* TextTrack) from addtrack
		// events fired by our own <track> being inserted (where
		// trackEl.track === t). Reactive — the manifest may surface
		// tracks at an unspecified time after loadedmetadata in
		// Safari's HLS implementation, so we can't gate on a
		// snapshot. (ACT-169.)
		this.videoTarget.textTracks.addEventListener("addtrack", (e) => {
			const t = e.track;
			if (t.kind !== "subtitles" && t.kind !== "captions") return;
			for (const trackEl of this.videoTarget.querySelectorAll("track")) {
				if (trackEl.label === t.label && trackEl.track !== t) {
					trackEl.remove();
					return;
				}
			}
		});
	}

	disconnect() {
		clearTimeout(this.#timerLoading);
	}

	dismiss() {
		this.element.remove();
	}

	handleControls(e) {
		//// Remove button states for fullscreen
		// const { controls: controlsElement } = elements;
		// if (controlsElement && event.type === 'enterfullscreen') {
		// 	controlsElement.pressed = false;
		// 	controlsElement.hover = false;
		// }

		// Show, then hide after a timeout unless another control event occurs
		const show = ["touchstart", "touchmove", "mousemove"].includes(e.type);
		let delay = 0;

		if (show) {
			this.#updateControlsVisibility(true);
			// Use longer timeout for touch devices
			delay = this.#isTouch ? 3000 : 2000;
		}

		// Clear timer
		clearTimeout(this.#timerControls);

		// Set new timer to prevent flicker when seeking
		this.#timerControls = setTimeout(() => this.#updateControlsVisibility(false), delay);
	}

	handleKey(e) {
		const { key, type, altKey, ctrlKey, metaKey, shiftKey } = e;
		const pressed = type === "keydown";
		const repeat = pressed && key === this.#lastKey;

		// Bail if a modifier key is set.
		if (altKey || ctrlKey || metaKey || shiftKey) return;
		if (!key) return;

		// Ignore key events when focused on editable elements.
		if (pressed) {
			const focused = document.activeElement;
			if (focused) {
				if (focused.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(focused.tagName)) return;
				// Don't intercept space on buttons/menu items (let them activate).
				if (key === " " && focused.matches("button, [role^=\"menuitem\"]")) return;
			}
		}

		// Prevent default for handled keys (e.g. prevent scrolling for arrows).
		const handled = [
			" ",
			"ArrowLeft",
			"ArrowUp",
			"ArrowRight",
			"ArrowDown",
			"0",
			"1",
			"2",
			"3",
			"4",
			"5",
			"6",
			"7",
			"8",
			"9",
			"a",
			"c",
			"f",
			"k",
			"l",
			"m",
		];
		if (pressed && handled.includes(key)) {
			e.preventDefault();
			e.stopPropagation();
		}

		if (pressed) {
			const video = this.videoTarget;

			switch (key) {
				// Seek to N/10 of duration.
				case "0":
				case "1":
				case "2":
				case "3":
				case "4":
				case "5":
				case "6":
				case "7":
				case "8":
				case "9":
					if (!repeat && this.#canSeek) {
						video.currentTime = (video.duration / 10) * parseInt(key, 10);
					}
					break;

				// Toggle play/pause.
				case " ":
				case "k":
					if (!repeat) this.togglePlay();
					break;

				// Volume up.
				case "ArrowUp":
					video.volume = Math.min(1, video.volume + 0.1);
					break;

				// Volume down.
				case "ArrowDown":
					video.volume = Math.max(0, video.volume - 0.1);
					break;

				// Toggle mute.
				case "m":
					if (!repeat) this.toggleMute();
					break;

				// Seek forward.
				case "ArrowRight":
					this.skipForward();
					break;

				// Seek backward.
				case "ArrowLeft":
					this.skipBackward();
					break;

				// Toggle fullscreen.
				case "f":
					if (!repeat) this.toggleFullscreen();
					break;

				// Toggle captions.
				case "c":
					if (!repeat) this.toggleCaptions();
					break;

				// Toggle audio menu.
				case "a":
					if (!repeat) this.toggleAudio();
					break;

				// Toggle loop.
				case "l":
					if (!repeat) video.loop = !video.loop;
					break;

				// Close video player.
				case "Escape":
					this.dismiss();
					break;
			}

			this.#lastKey = key;
		} else {
			this.#lastKey = null;
		}
	}

	handlePlaying(e) {
		this.playingValue = this.videoTarget.playing;
		this.pausedValue = this.videoTarget.paused;
		this.stoppedValue = this.videoTarget.stopped;

		// Only update controls on non timeupdate events
		if (e.type === "timeupdate") {
			return;
		}
		this.#updateControlsVisibility();
	}

	#updateControlsVisibility(recentInteraction = false) {
		this.#recentInteraction = recentInteraction;

		// Don't hide controls if a touch-device user recently seeked.
		// (Must be limited to touch devices, or it occasionally prevents
		// desktop controls from hiding.)
		this.#recentTouchSeek = this.#isTouch && this.#lastSeekTime + 2000 > Date.now();

		this.#setControlsVisibility();
	}

	#setControlsVisibility() {
		const show = this.#doShowControls;
		this.hideControlsValue = !show;

		//// Close menu
		// if (
		// 	hiding
		// 	&& is.array(this.config.controls)
		// 	&& this.config.controls.includes('settings')
		// 	&& !is.empty(this.config.settings)
		// ) {
		// 	controls.toggleMenu.call(this, false);
		// }

		return show;
	}

	toggleHarlow() {
		this.harlowValue = !this.harlowValue;
	}

	// --- Quality menu ---

	toggleQualityMenu() {
		this.qualityMenuOpenValue = !this.qualityMenuOpenValue;
	}

	setQuality(e) {
		const url = e.params.url;
		const label = e.params.label;
		const video = this.videoTarget;

		const currentTime = video.currentTime;
		const wasPlaying = !video.paused;

		// Switch the source. Drop any <track> children we cloned from
		// the captions template — they were tied to the old manifest's
		// TextTrack list, and the new source's #ensureSubtitleTracks
		// pass needs a clean slate to decide whether to re-insert.
		for (const te of video.querySelectorAll("track")) te.remove();
		this.#subtitleTracksDecided = false;
		const source = video.querySelector("source");
		if (source) {
			source.src = url;
		} else {
			video.src = url;
		}
		video.load();

		// Restore position once enough data is available.
		const restore = () => {
			video.currentTime = currentTime;
			if (wasPlaying) video.play();
			video.removeEventListener("loadedmetadata", restore);
		};
		video.addEventListener("loadedmetadata", restore);

		this.currentQualityValue = label;
		this.qualityMenuOpenValue = false;

		// Update active state on menu items.
		for (const btn of this.qualityMenuTarget.querySelectorAll("button")) {
			if (btn.dataset.playerLabelParam === label) {
				btn.setAttribute("data-active", "");
			} else {
				btn.removeAttribute("data-active");
			}
		}
	}

	togglePlay() {
		const video = this.videoTarget;
		if (video.paused) {
			video.play();
		} else {
			video.pause();
		}
	}

	toggleFullscreen() {
		if (document.fullscreenElement) {
			document.exitFullscreen();
		} else {
			this.element.requestFullscreen();
		}
	}

	// 'c' hotkey opens the captions menu — the user picks a track from
	// the popover. v1: discoverable, no hidden state about which track
	// was last shown.
	toggleCaptions() {
		this.toggleCaptionsMenu();
	}

	toggleCaptionsMenu() {
		this.captionsMenuOpenValue = !this.captionsMenuOpenValue;
	}

	setSubtitle(e) {
		const id = e.params.subId;
		const video = this.videoTarget;

		for (const t of video.textTracks) {
			if (t.kind === "subtitles" || t.kind === "captions") {
				t.mode = "disabled";
			}
		}

		// TextTrack.label carries the SubtitleTrack ID (set as
		// EXT-X-MEDIA NAME and as the captions-template's label
		// attribute) — match on it directly.
		if (id !== "") {
			for (const t of video.textTracks) {
				if (t.kind !== "subtitles" && t.kind !== "captions") continue;
				if (t.label === id) {
					t.mode = "showing";
					break;
				}
			}
		}

		this.currentSubtitleValue = id;
		this.captionsMenuOpenValue = false;

		for (const btn of this.captionsMenuTarget.querySelectorAll("button")) {
			if (btn.dataset.playerSubIdParam === id) {
				btn.setAttribute("data-active", "");
			} else {
				btn.removeAttribute("data-active");
			}
		}
	}

	// 'a' hotkey opens the audio-track menu — the user picks a
	// rendition from the popover. Mirrors toggleCaptions.
	toggleAudio() {
		this.toggleAudioMenu();
	}

	toggleAudioMenu() {
		this.audioMenuOpenValue = !this.audioMenuOpenValue;
	}

	setAudio(e) {
		const id = e.params.audioId;
		const video = this.videoTarget;

		// Native audioTracks: setting enabled=true on one track
		// exclusively selects it (the spec says siblings flip to false
		// and the change event fires). Walk and assign for clarity.
		// Track .label is the manifest's EXT-X-MEDIA NAME, which we
		// set to the AudioRendition ID — match by id directly.
		if (!video.audioTracks) return;
		for (const t of video.audioTracks) {
			t.enabled = t.label === id;
		}

		this.currentAudioValue = id;
		this.audioMenuOpenValue = false;

		for (const btn of this.audioMenuTarget.querySelectorAll("button")) {
			if (btn.dataset.playerAudioIdParam === id) {
				btn.setAttribute("data-active", "");
			} else {
				btn.removeAttribute("data-active");
			}
		}
	}

	toggleMute() {
		this.videoTarget.muted = !this.videoTarget.muted;
	}

	handleVolumeInput(e) {
		const value = parseFloat(e.target.value);
		this.videoTarget.volume = value;
		this.#setVolumeFill(value);
	}

	handleVolumeWheel(e) {
		// Detect "natural" scroll (OS X Safari)
		const inverted = e.webkitDirectionInvertedFromDevice;
		const [x, y] = [e.deltaX, -e.deltaY].map(v => (inverted ? -v : v));
		const direction = Math.sign(Math.abs(x) > Math.abs(y) ? x : y);

		// Change volume by 2%
		const video = this.videoTarget;
		video.volume = Math.max(0, Math.min(1, video.volume + direction / 50));

		// Don't break page scrolling at min/max
		if ((direction === 1 && video.volume < 1) || (direction === -1 && video.volume > 0)) {
			e.preventDefault();
		}
	}

	handleVolume() {
		// Sync slider UI when the video's volume changes (e.g. from mute toggle or keyboard).
		// When muted, show the slider at 0; otherwise show the actual volume.
		const video = this.videoTarget;
		const vol = video.muted ? 0 : video.volume;
		this.volumeTarget.value = vol;
		this.#setVolumeFill(vol);
	}

	#setVolumeFill(vol) {
		this.volumeTarget.style.setProperty("--value", `${vol * 100}%`);
	}

	// --- Seek bar handlers ---

	handleSeek(e) {
		const seek = e.currentTarget;
		// Use seek-value if set (for tooltip consistency), otherwise use value.
		let seekTo = seek.getAttribute("seek-value");
		if (seekTo == null) seekTo = seek.value;
		seek.removeAttribute("seek-value");

		const video = this.videoTarget;
		if (video.duration) {
			video.currentTime = (seekTo / seek.max) * video.duration;
		}
		this.#setSeekFill(seekTo);
	}

	handleSeekMouse(e) {
		// Set seek-value attribute so handleSeek uses the mouse position
		// rather than the input value (matches tooltip time).
		// Use clientX (viewport-relative) since getBoundingClientRect is also viewport-relative.
		const rect = this.progressTarget.getBoundingClientRect();
		const percent = (100 / rect.width) * (e.clientX - rect.left);
		e.currentTarget.setAttribute("seek-value", Math.max(0, Math.min(100, percent)));
	}

	handleSeekPause(e) {
		const seek = e.currentTarget;
		const attr = "data-play-on-seeked";

		// Only handle arrow keys for keyboard events.
		if (e instanceof KeyboardEvent && !["ArrowLeft", "ArrowRight"].includes(e.key)) return;

		// Record seek time so controls stay visible after seeking.
		this.#lastSeekTime = Date.now();

		const wasPlaying = seek.hasAttribute(attr);
		const done = ["mouseup", "touchend", "keyup"].includes(e.type);
		this.#userSeeking = !done;

		// If done seeking and was playing, resume playback.
		if (wasPlaying && done) {
			seek.removeAttribute(attr);
			this.videoTarget.play();
		} else if (!done && !this.videoTarget.paused) {
			// Starting a seek while playing — pause and remember.
			seek.setAttribute(attr, "");
			this.videoTarget.pause();
		}
	}

	handleSeekTooltip(e) {
		const tooltip = this.seekTooltipTarget;

		// Hide tooltip on touch devices and on mouseleave.
		if (this.#isTouch || e.type === "mouseleave") {
			tooltip.style.opacity = "0";
			return;
		}

		const duration = this.videoTarget.duration;
		if (!duration) return;

		// Use clientX (viewport-relative) to match getBoundingClientRect.
		const rect = this.progressTarget.getBoundingClientRect();
		const percent = Math.max(0, Math.min(100, (100 / rect.width) * (e.clientX - rect.left)));
		const time = (percent / 100) * duration;

		tooltip.textContent = this.#formatTime(time);
		tooltip.style.left = `${percent}%`;
		tooltip.style.opacity = "1";
	}

	skipBackward() {
		if (!this.#canSeek) return;
		const video = this.videoTarget;
		video.currentTime = Math.max(0, video.currentTime - 10);
	}

	skipForward() {
		if (!this.#canSeek) return;
		const video = this.videoTarget;
		video.currentTime = Math.min(video.duration, video.currentTime + 10);
	}

	// Safari (with native HLS) can have video.duration available from the
	// manifest while readyState is still too low to actually seek. Setting
	// currentTime in that state permanently breaks Safari's media pipeline.
	// HAVE_CURRENT_DATA (2) means the browser has actual media data, not
	// just metadata, so seeking is safe.
	get #canSeek() {
		return this.videoTarget.readyState >= 2;
	}

	handleLoading(e) {
		const loading = ["stalled", "waiting"].includes(e.type);

		clearTimeout(this.#timerLoading);

		// 250ms delay when entering loading to prevent flicker during seeks;
		// immediate when leaving loading.
		this.#timerLoading = setTimeout(() => {
			this.loadingValue = loading;
			this.#updateControlsVisibility();
		}, loading ? 250 : 0);
	}

	// --- Time/duration/buffer handlers (from video events) ---

	handleTimeUpdate(e) {
		const video = this.videoTarget;
		if (!video.duration) return;

		const currentTime = video.currentTime;
		const duration = video.duration;
		const percent = (currentTime / duration) * 100;

		// Only move the seek slider on timeupdate, not seeking/seeked
		// (during seeking/seeked the user is controlling the slider).
		// #userSeeking gates updates while the user is dragging.
		// (We don't check video.seeking here — it can get permanently
		// stuck on HLS streams if currentTime is set before segments load.)
		const isTimeUpdate = e && e.type === "timeupdate";
		if (isTimeUpdate && !this.#userSeeking) {
			this.seekTarget.value = percent;
			this.#setSeekFill(percent);
		}

		// Always update the time display regardless of event type.
		this.currentTimeTarget.textContent = this.#formatTime(currentTime);
	}

	handleDurationChange() {
		const duration = this.videoTarget.duration;
		if (!duration || !isFinite(duration)) return;

		// Hide time/progress for live streams (duration >= 2^32).
		if (duration >= 2 ** 32) {
			this.currentTimeTarget.hidden = true;
			this.progressTarget.hidden = true;
			return;
		}

		// Update aria-valuemax on the seek input for accessibility.
		this.seekTarget.setAttribute("aria-valuemax", duration);
		this.durationTarget.textContent = this.#formatTime(duration);
	}

	// loadedmetadata is the spec-guaranteed point at which the
	// browser's HLS implementation has populated textTracks and
	// audioTracks from the manifest. Earlier events (durationchange,
	// loadeddata) can fire before parsing is complete, so any logic
	// that probes the track lists must wait for this one — otherwise
	// the captions-template fallback can clone tracks that Safari is
	// about to surface natively (ACT-169).
	handleLoadedMetadata() {
		this.#ensureSubtitleTracks();
		this.#filterCaptionsMenu();
		this.#syncSubtitleFromTracks();

		this.#filterAudioMenu();
		this.#applyAudioSelection();
	}

	// Hide menu entries whose id doesn't match any TextTrack the
	// browser surfaced — picks route through TextTrack.mode, so an
	// entry without a matching track wouldn't render anything. ASS
	// sources downconvert to WebVTT and reach textTracks the same way
	// as everything else. Track .label carries the SubtitleTrack ID
	// (set as EXT-X-MEDIA NAME and as the captions-template's label).
	#filterCaptionsMenu() {
		const ids = new Set();
		for (const t of this.videoTarget.textTracks) {
			if (t.kind === "subtitles" || t.kind === "captions") {
				ids.add(t.label);
			}
		}
		for (const btn of this.captionsMenuTarget.querySelectorAll("button")) {
			if (btn.dataset.playerSubIdParam === "") continue;
			btn.hidden = !ids.has(btn.dataset.playerSubIdParam);
		}
	}

	// Clone the captions <template>'s <track> elements into <video>
	// so the browser fetches and renders the WebVTT files directly.
	// This is the fallback for browsers whose HLS implementation
	// doesn't surface the manifest's SUBTITLES group via textTracks
	// (Chromium #383582114), and it also surfaces FORCED=YES tracks
	// that Safari's native HLS hides from the textTracks API
	// (ACT-170). When the manifest *does* surface a track, an
	// addtrack listener installed in connect() removes the
	// corresponding cloned <track> to avoid duplicates (ACT-169).
	// Decided once per source; setQuality clears the gate.
	#ensureSubtitleTracks() {
		if (this.#subtitleTracksDecided) return;
		this.#subtitleTracksDecided = true;
		if (!this.hasCaptionsTemplateTarget) return;
		this.videoTarget.appendChild(
			this.captionsTemplateTarget.content.cloneNode(true),
		);
	}

	// Hide audio-menu entries whose id doesn't match any audioTracks
	// entry the browser surfaced. If audioTracks is empty (manifest
	// not yet parsed), entries stay visible — the next handleDuration
	// pass will prune. Track .label carries the AudioRendition ID
	// (set as EXT-X-MEDIA NAME in buildMVPlaylist).
	#filterAudioMenu() {
		if (!this.videoTarget.audioTracks) return;
		if (this.videoTarget.audioTracks.length === 0) return;
		const ids = new Set();
		for (const t of this.videoTarget.audioTracks) ids.add(t.label);
		for (const btn of this.audioMenuTarget.querySelectorAll("button")) {
			btn.hidden = !ids.has(btn.dataset.playerAudioIdParam);
		}
	}

	// On each loaded manifest, force the chosen audio track:
	//   - no user pick yet: the publisher's DEFAULT (the menu item
	//     marked data-active server-side). Safari's native HLS picks
	//     by system locale and ignores DEFAULT=YES; we override that
	//     to honor the publisher's source-mux order (ACT-145 design).
	//   - user has picked: re-enable that track. Quality switches
	//     produce a new AudioTracks list whose default may not match
	//     the user's choice; without re-applying, switching quality
	//     would silently revert the audio selection.
	#applyAudioSelection() {
		if (!this.videoTarget.audioTracks) return;
		if (this.currentAudioValue === "") {
			const btn = this.audioMenuTarget.querySelector("button[data-active]");
			if (!btn) return;
			this.currentAudioValue = btn.dataset.playerAudioIdParam;
		}
		const id = this.currentAudioValue;
		for (const t of this.videoTarget.audioTracks) {
			t.enabled = t.label === id;
		}
	}

	// Reflect a textTrack the HLS player auto-enabled (e.g. Safari with
	// DEFAULT=YES) into our menu state. No-op once the user has made an
	// explicit pick — currentSubtitle is the source of truth from then on.
	#syncSubtitleFromTracks() {
		if (this.currentSubtitleValue !== "") return;
		for (const t of this.videoTarget.textTracks) {
			if (t.kind !== "subtitles" && t.kind !== "captions") continue;
			if (t.mode !== "showing") continue;
			for (const btn of this.captionsMenuTarget.querySelectorAll("button")) {
				if (btn.dataset.playerSubIdParam !== t.label) continue;
				this.currentSubtitleValue = btn.dataset.playerSubIdParam;
				for (const b of this.captionsMenuTarget.querySelectorAll("button")) {
					if (b === btn) b.setAttribute("data-active", "");
					else b.removeAttribute("data-active");
				}
				return;
			}
		}
	}

	handleProgress() {
		const video = this.videoTarget;

		let buffered = 0;
		if (video.buffered && video.buffered.length > 0 && video.duration) {
			buffered = (video.buffered.end(video.buffered.length - 1) / video.duration) * 100;
		}

		this.bufferTarget.value = buffered;
	}

	// --- Seek bar helpers ---

	#setSeekFill(percent) {
		this.seekTarget.style.setProperty("--value", `${percent}%`);
	}

	#formatTime(seconds) {
		if (!isFinite(seconds) || seconds < 0) return "0:00";
		// Always show hours if the total duration is >= 1 hour,
		// so the display width stays stable (e.g. "0:05:30" not "5:30").
		const forceHours = this.videoTarget.duration >= 3600;
		const hrs = Math.floor(seconds / 3600);
		const mins = Math.floor((seconds % 3600) / 60);
		const secs = Math.floor(seconds % 60);
		if (hrs > 0 || forceHours) {
			return `${hrs}:${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
		}
		return `${mins}:${secs.toString().padStart(2, "0")}`;
	}

	get #doShowControls() {
		if (this.#harlowMode) {
			return false;
		}
		// Show controls if recentInteraction, loading, paused,
		// button active, or recent touch seek, otherwise hide.
		return this.#recentInteraction
			|| this.loadingValue
			|| this.videoTarget.paused
			// controlsElement.pressed ||
			// controlsElement.hover ||
			|| this.#recentTouchSeek;
	}
}
