import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static targets = [
		"video",
		"controls",
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
		videoId: String,
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
		// Empty in default builds; populated only by -tags jassub.
		// A non-empty hostUrl unlocks the libass-via-WASM branch
		// of #applySubtitleSelection for ass/ssa picks.
		jassubHostUrl: String,
		jassubWorkerUrl: String,
		jassubWasmUrl: String,
		jassubFontUrl: String,
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
	#pendingSeek = null;
	#jassubInstance = null;
	#jassubActiveId = null;
	// Bumped on every teardown / new setup to invalidate stale
	// async work from the previous selection (the dynamic-import
	// of the host module is the slow leg).
	#jassubGen = 0;

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
		// snapshot. (ACT-169.) Re-apply the seeded subtitle after
		// each addtrack so a late-arriving manifest track is forced
		// into the user's chosen mode.
		this.videoTarget.textTracks.addEventListener("addtrack", (e) => {
			const t = e.track;
			if (t.kind !== "subtitles" && t.kind !== "captions") return;
			for (const trackEl of this.videoTarget.querySelectorAll("track")) {
				if (trackEl.label === t.label && trackEl.track !== t) {
					trackEl.remove();
					break;
				}
			}
			this.#applySubtitleSelection();
		});

		// While jassub is rendering, no native textTrack should be
		// showing — the WebVTT version of the same subtitle would
		// paint over libass. Safari's HLS layer can flip a manifest
		// track with DEFAULT=YES to "showing" outside of addtrack
		// timing (it's been observed when the player opens with the
		// ASS sub already preselected via the URL ?s= query), so
		// we listen for any mode change and re-disable on the spot
		// when jassub is active. Idempotent: setting a track that's
		// already disabled to "disabled" doesn't fire change again.
		this.videoTarget.textTracks.addEventListener("change", () => {
			if (!this.#jassubInstance) return;
			for (const t of this.videoTarget.textTracks) {
				if (t.kind !== "subtitles" && t.kind !== "captions") continue;
				if (t.mode !== "disabled") t.mode = "disabled";
			}
		});

		// When the manifest surfaces an audio AudioTrack, re-apply
		// the chosen audio selection. Safari fires loadedmetadata
		// before audioTracks is populated for native HLS, so applying
		// once in handleLoadedMetadata would race the empty list and
		// Safari would fall back to system-locale-based audio
		// (ACT-145). Reactive here, like the textTracks listener
		// above. Chrome lacks the API entirely; .audioTracks is
		// undefined and there's nothing to listen on.
		if (this.videoTarget.audioTracks) {
			this.videoTarget.audioTracks.addEventListener("addtrack", () => {
				this.#applyAudioSelection();
			});
		}

		// The video element may have already fired loadedmetadata
		// before Stimulus wired up its action listener — readyState>=1
		// means we missed the event. Replay the handler so seeded
		// audio and subtitle selections take effect on first paint.
		if (this.videoTarget.readyState >= 1) {
			this.handleLoadedMetadata();
		}
	}

	disconnect() {
		clearTimeout(this.#timerLoading);
		this.#destroyJassub();
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

				// Close the open menu, or otherwise close the player.
				case "Escape":
					if (this.#anyMenuOpen) {
						this.#closeAllMenus();
					} else {
						this.dismiss();
					}
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
		const qualityId = e.params.qualityId;
		const video = this.videoTarget;

		// Safari switches audio via the audioTracks API, so its URL
		// only carries the quality pin (the full audio group is
		// preserved in the manifest, and the API stays usable).
		// Chrome has no audioTracks API, so audio must be source-
		// swapped too — include the current audio pin in the URL so
		// the user's audio choice survives the quality switch.
		const audioPin = video.audioTracks ? "" : this.currentAudioValue;
		this.#sourceSwap(this.#composedURL(qualityId, audioPin));

		this.currentQualityValue = qualityId;
		this.qualityMenuOpenValue = false;

		for (const btn of this.qualityMenuTarget.querySelectorAll("button")) {
			if (btn.dataset.playerQualityIdParam === qualityId) {
				btn.setAttribute("data-active", "");
			} else {
				btn.removeAttribute("data-active");
			}
		}
	}

	// composedURL builds the player's MV playlist URL with optional
	// quality and audio pins. Empty values are omitted from the
	// query string. This is the one URL the player ever requests for
	// the multivariant playlist; the server narrows the variant /
	// audio set per the parameters.
	#composedURL(qualityId, audioId) {
		let url = `/-/plr/${this.videoIdValue}.m3u8`;
		const params = [];
		if (qualityId) params.push(`q=${encodeURIComponent(qualityId)}`);
		if (audioId) params.push(`a=${encodeURIComponent(audioId)}`);
		if (params.length > 0) url += "?" + params.join("&");
		return url;
	}

	// sourceSwap replaces the <video> source URL and re-loads,
	// restoring the previous play state and position once the new
	// source has data. Drops any <track> children cloned from the
	// captions template (they were tied to the old manifest's
	// TextTrack list) and resets the subtitle-tracks gate so
	// #ensureSubtitleTracks can decide afresh for the new source.
	// Tears down any active jassub renderer too — the subsequent
	// loadedmetadata triggers #applySubtitleSelection, which will
	// recreate it when currentSubtitle is still an ASS pick.
	//
	// The position restore is deferred to loadeddata, not applied at
	// loadedmetadata: setting currentTime before HAVE_CURRENT_DATA
	// permanently wedges Safari's requestVideoFrameCallback delivery
	// for this element. Playback continues off the normal pipeline,
	// but jassub renders solely off rVFC, so it freezes and the ASS
	// overlay vanishes for the rest of the session (ACT-220). The
	// seek rides the same #pendingSeek path the seekbar uses (ACT-171);
	// play() resumes at loadedmetadata, which is also what prompts
	// Safari to start loading data so loadeddata can fire.
	#sourceSwap(url) {
		const video = this.videoTarget;
		const currentTime = video.currentTime;
		const wasPlaying = !video.paused;

		for (const te of video.querySelectorAll("track")) te.remove();
		this.#subtitleTracksDecided = false;
		this.#destroyJassub();

		const source = video.querySelector("source");
		if (source) {
			source.src = url;
		} else {
			video.src = url;
		}
		video.load();

		this.#pendingSeek = currentTime;
		if (wasPlaying) {
			const resume = () => {
				video.play();
				video.removeEventListener("loadedmetadata", resume);
			};
			video.addEventListener("loadedmetadata", resume);
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

	closeMenusOnClick(e) {
		if (!this.#anyMenuOpen) return;
		// Close every open menu except the one whose wrapper was
		// clicked into. Letting the click bubble to that menu's own
		// toggle (or option) lets it do the right thing — toggling
		// off if the click was its own toggle button, or running an
		// option's setter (which closes the menu itself).
		const insideMenu = e.target.closest("[data-player-menu]")?.dataset.playerMenu ?? null;
		for (const key of ["quality", "captions", "audio"]) {
			if (key === insideMenu) continue;
			const valueProp = key + "MenuOpenValue";
			if (this[valueProp]) this[valueProp] = false;
		}
		// Suppress togglePlay specifically when the click was on the
		// .v-player-controls backdrop (i.e., on the video itself) —
		// dismissing the menu by clicking the video shouldn't also
		// flip playback. Clicks on real controls (which have their
		// own targets) propagate normally.
		if (e.target === this.controlsTarget) {
			e.stopPropagation();
		}
	}

	get #anyMenuOpen() {
		return this.qualityMenuOpenValue
			|| this.captionsMenuOpenValue
			|| this.audioMenuOpenValue;
	}

	#closeAllMenus() {
		this.qualityMenuOpenValue = false;
		this.captionsMenuOpenValue = false;
		this.audioMenuOpenValue = false;
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
		this.currentSubtitleValue = id;
		this.captionsMenuOpenValue = false;

		// #applySubtitleSelection reads the menu's data attrs on
		// the active button to decide between the textTracks path
		// and the jassub path, so it stays the single source of
		// truth across user clicks, addtrack events, and post-
		// sourceSwap loadedmetadata.
		this.#applySubtitleSelection();

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

		if (video.audioTracks) {
			// Safari: native audioTracks API. Setting enabled=true
			// on one track exclusively selects it (siblings flip to
			// false and the change event fires). Track .label is the
			// manifest's EXT-X-MEDIA NAME — which we set to the
			// AudioRendition ID — so match by id directly.
			for (const t of video.audioTracks) {
				t.enabled = t.label === id;
			}
		} else {
			// Chrome: no audioTracks API. Source-swap to a combined-
			// pin URL that preserves the current quality choice.
			this.#sourceSwap(this.#composedURL(this.currentQualityValue, id));
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
			const targetTime = (seekTo / seek.max) * video.duration;
			if (this.#canSeek) {
				video.currentTime = targetTime;
			} else {
				// Safari's native HLS pipeline breaks if currentTime is
				// set before HAVE_CURRENT_DATA. Stash the seek and let
				// handleLoadedData replay it (ACT-171).
				this.#pendingSeek = targetTime;
			}
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
		this.#applySubtitleSelection();

		this.#applyAudioSelection();

		// Enable the seekbar now that duration is known. We can't wait
		// for loadeddata — Safari with native HLS may not reach
		// HAVE_CURRENT_DATA until play() — so we accept clicks earlier
		// and defer the seek via #pendingSeek (ACT-171).
		this.seekTarget.disabled = false;
	}

	// loadeddata fires when readyState reaches HAVE_CURRENT_DATA — the
	// earliest point at which Safari's native HLS pipeline tolerates
	// setting currentTime. Apply any seek the user made in the gap
	// between loadedmetadata and now (ACT-171).
	handleLoadedData() {
		if (this.#pendingSeek == null) return;
		this.videoTarget.currentTime = this.#pendingSeek;
		this.#pendingSeek = null;
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

	// On each loaded manifest, force currentAudio onto the
	// audioTracks list. Safari ignores HLS DEFAULT=YES and picks by
	// system locale, so we override it (ACT-145). Re-applying matters
	// across quality switches too: the new AudioTrackList may not
	// preserve our enabled flag. Chrome lacks the API entirely; its
	// playing audio is whatever the manifest carries, which the
	// theater pins via pin_audio=1 server-side.
	#applyAudioSelection() {
		if (!this.videoTarget.audioTracks) return;
		const id = this.currentAudioValue;
		for (const t of this.videoTarget.audioTracks) {
			t.enabled = t.label === id;
		}
	}

	// On each loaded manifest (or addtrack), force currentSubtitle
	// onto the renderer. Seed comes from the player URL ("" = Off).
	// Forcing matters in Safari, which auto-enables a manifest
	// track with DEFAULT=YES regardless of our intent — without
	// this, "Off" wouldn't actually be off.
	//
	// In a jassub-tagged build, ass/ssa picks route through
	// #startJassub instead of textTracks; this method is the
	// branch point. The codec and original-format URL come from
	// the menu button's data attrs (see playerCaptionsItem in
	// view/player.go), so a single read keeps state aligned
	// across user clicks and reactive (addtrack / loadedmetadata)
	// re-applies.
	#applySubtitleSelection() {
		const id = this.currentSubtitleValue;
		const btn = id
			? this.captionsMenuTarget.querySelector(
				`button[data-player-sub-id-param="${CSS.escape(id)}"]`,
			)
			: null;
		const codec = btn?.dataset.playerSubCodecParam || "";
		const originalUrl = btn?.dataset.playerSubOriginalParam || "";

		const useJassub = id !== ""
			&& this.jassubHostUrlValue !== ""
			&& (codec === "ass" || codec === "ssa")
			&& originalUrl !== "";

		if (useJassub) {
			// A native track for the same id would render the
			// lossy WebVTT alongside libass — disable everything.
			for (const t of this.videoTarget.textTracks) {
				if (t.kind === "subtitles" || t.kind === "captions") {
					t.mode = "disabled";
				}
			}
			if (this.#jassubActiveId !== id) {
				this.#startJassub(originalUrl, id);
			}
			return;
		}

		this.#destroyJassub();
		for (const t of this.videoTarget.textTracks) {
			if (t.kind !== "subtitles" && t.kind !== "captions") continue;
			t.mode = id !== "" && t.label === id ? "showing" : "disabled";
		}
	}

	// Dynamic-import the jassub host module and instantiate it
	// over the video. The host module is bundled at vendor time
	// (web/static/jassub/gen.go), so the import resolves to a
	// single self-contained ESM file.
	async #startJassub(subUrl, id) {
		const gen = ++this.#jassubGen;
		this.#destroyJassub();
		// #destroyJassub bumps #jassubGen; restore our token so
		// only later switches invalidate this attempt.
		this.#jassubGen = gen;

		let mod;
		try {
			mod = await import(this.jassubHostUrlValue);
		} catch {
			return;
		}
		if (gen !== this.#jassubGen) return;

		this.#jassubInstance = new mod.default({
			video: this.videoTarget,
			subUrl,
			workerUrl: this.jassubWorkerUrlValue,
			wasmUrl: this.jassubWasmUrlValue,
			modernWasmUrl: this.jassubWasmUrlValue,
			// Hand-supply the Liberation Sans fallback so libass has
			// a font to open — without it nothing renders. jassub
			// would otherwise resolve `./default.woff2` relative to
			// the host module, which fails since our handler serves
			// digest-cache-busted URLs.
			availableFonts: { "liberation sans": this.jassubFontUrlValue },
			// Don't try to discover local fonts via the Local Font
			// Access API — act3 doesn't ship them and the discovery
			// path forces a host→worker proxy roundtrip that throws
			// on Chromium variants without the permission.
			queryFonts: false,
		});
		this.#jassubActiveId = id;
	}

	#destroyJassub() {
		this.#jassubGen++;
		if (this.#jassubInstance) {
			this.#jassubInstance.destroy();
			this.#jassubInstance = null;
		}
		this.#jassubActiveId = null;
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
		// Show controls if recentInteraction, loading, paused, a menu is
		// open, the cursor is over the controls, or recent touch seek,
		// otherwise hide.
		return this.#recentInteraction
			|| this.loadingValue
			|| this.videoTarget.paused
			|| this.#anyMenuOpen
			|| this.#controlsHovered
			|| this.#recentTouchSeek;
	}

	get #controlsHovered() {
		return !!this.controlsTarget.querySelector(
			".v-player-overlay-top:hover, .v-player-overlay-bottom:hover",
		);
	}
}
