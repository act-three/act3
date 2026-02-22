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
	];
	static values = {
		iconUrl: String,
		title: String,
		playing: Boolean,
		paused: Boolean,
		stopped: Boolean,
		harlow: Boolean,
		hideControls: Boolean,
		loading: Boolean,
		currentQuality: String,
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

	connect() {
	}

	disconnect() {
		clearTimeout(this.#timerLoading);
	}

	dismiss() {
		this.element.remove();
	}

	handleControls(e) {
		//// Remove button states for fullscreen
		//const { controls: controlsElement } = elements;
		//if (controlsElement && event.type === 'enterfullscreen') {
		//	controlsElement.pressed = false;
		//	controlsElement.hover = false;
		//}

		// Show, then hide after a timeout unless another control event occurs
		const show = ['touchstart', 'touchmove', 'mousemove'].includes(e.type);
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
		const pressed = type === 'keydown';
		const repeat = pressed && key === this.#lastKey;

		// Bail if a modifier key is set.
		if (altKey || ctrlKey || metaKey || shiftKey) return;
		if (!key) return;

		// Ignore key events when focused on editable elements.
		if (pressed) {
			const focused = document.activeElement;
			if (focused) {
				if (focused.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(focused.tagName)) return;
				// Don't intercept space on buttons/menu items (let them activate).
				if (key === ' ' && focused.matches('button, [role^="menuitem"]')) return;
			}
		}

		// Prevent default for handled keys (e.g. prevent scrolling for arrows).
		const handled = [
			' ', 'ArrowLeft', 'ArrowUp', 'ArrowRight', 'ArrowDown',
			'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
			'c', 'f', 'k', 'l', 'm',
		];
		if (pressed && handled.includes(key)) {
			e.preventDefault();
			e.stopPropagation();
		}

		if (pressed) {
			const video = this.videoTarget;

			switch (key) {
				// Seek to N/10 of duration.
				case '0': case '1': case '2': case '3': case '4':
				case '5': case '6': case '7': case '8': case '9':
					if (!repeat && this.#canSeek) {
						video.currentTime = (video.duration / 10) * parseInt(key, 10);
					}
					break;

				// Toggle play/pause.
				case ' ':
				case 'k':
					if (!repeat) this.togglePlay();
					break;

				// Volume up.
				case 'ArrowUp':
					video.volume = Math.min(1, video.volume + 0.1);
					break;

				// Volume down.
				case 'ArrowDown':
					video.volume = Math.max(0, video.volume - 0.1);
					break;

				// Toggle mute.
				case 'm':
					if (!repeat) this.toggleMute();
					break;

				// Seek forward.
				case 'ArrowRight':
					this.skipForward();
					break;

				// Seek backward.
				case 'ArrowLeft':
					this.skipBackward();
					break;

				// Toggle fullscreen.
				case 'f':
					if (!repeat) this.toggleFullscreen();
					break;

				// Toggle captions.
				case 'c':
					if (!repeat) this.toggleCaptions();
					break;

				// Toggle loop.
				case 'l':
					if (!repeat) { video.loop = !video.loop; }
					break;

				// Close video player.
				case 'Escape':
					this.dismiss()
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
		if (e.type === 'timeupdate') {
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
		//if (
		//	hiding
		//	&& is.array(this.config.controls)
		//	&& this.config.controls.includes('settings')
		//	&& !is.empty(this.config.settings)
		//) {
		//	controls.toggleMenu.call(this, false);
		//}

		return show;
	}

	toggleHarlow() {
		this.harlowValue = !this.harlowValue;
	}

	// --- Quality menu ---

	toggleQualityMenu() {
		this.qualityMenuTarget.classList.toggle("hidden");
	}

	setQuality(e) {
		const url = e.params.url;
		const label = e.params.label;
		const video = this.videoTarget;

		const currentTime = video.currentTime;
		const wasPlaying = !video.paused;

		// Switch the source.
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
		this.qualityMenuTarget.classList.add("hidden");

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

	toggleCaptions() {
		const video = this.videoTarget;
		for (const track of video.textTracks) {
			if (track.kind === 'captions' || track.kind === 'subtitles') {
				track.mode = track.mode === 'showing' ? 'hidden' : 'showing';
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
		this.volumeTarget.style.setProperty('--value', `${vol * 100}%`);
	}

	// --- Seek bar handlers ---

	handleSeek(e) {
		const seek = e.currentTarget;
		// Use seek-value if set (for tooltip consistency), otherwise use value.
		let seekTo = seek.getAttribute('seek-value');
		if (seekTo == null) seekTo = seek.value;
		seek.removeAttribute('seek-value');

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
		e.currentTarget.setAttribute('seek-value', Math.max(0, Math.min(100, percent)));
	}

	handleSeekPause(e) {
		const seek = e.currentTarget;
		const attr = 'data-play-on-seeked';

		// Only handle arrow keys for keyboard events.
		if (e instanceof KeyboardEvent && !['ArrowLeft', 'ArrowRight'].includes(e.key)) return;

		// Record seek time so controls stay visible after seeking.
		this.#lastSeekTime = Date.now();

		const wasPlaying = seek.hasAttribute(attr);
		const done = ['mouseup', 'touchend', 'keyup'].includes(e.type);
		this.#userSeeking = !done;

		// If done seeking and was playing, resume playback.
		if (wasPlaying && done) {
			seek.removeAttribute(attr);
			this.videoTarget.play();
		} else if (!done && !this.videoTarget.paused) {
			// Starting a seek while playing — pause and remember.
			seek.setAttribute(attr, '');
			this.videoTarget.pause();
		}
	}

	handleSeekTooltip(e) {
		const tooltip = this.seekTooltipTarget;

		// Hide tooltip on touch devices and on mouseleave.
		if (this.#isTouch || e.type === 'mouseleave') {
			tooltip.style.opacity = '0';
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
		tooltip.style.opacity = '1';
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
		const loading = ['stalled', 'waiting'].includes(e.type);

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
		const isTimeUpdate = e && e.type === 'timeupdate';
		if (isTimeUpdate && !this.#userSeeking) {
			this.seekTarget.value = percent;
			this.#setSeekFill(percent);
		}

		// Always update the time display regardless of event type.
		this.currentTimeTarget.textContent = this.#formatTime(currentTime);
	}

	handleDuration() {
		const duration = this.videoTarget.duration;
		if (!duration || !isFinite(duration)) return;

		// Hide time/progress for live streams (duration >= 2^32).
		if (duration >= 2 ** 32) {
			this.currentTimeTarget.hidden = true;
			this.progressTarget.hidden = true;
			return;
		}

		// Update aria-valuemax on the seek input for accessibility.
		this.seekTarget.setAttribute('aria-valuemax', duration);

		this.durationTarget.textContent = this.#formatTime(duration);
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
		this.seekTarget.style.setProperty('--value', `${percent}%`);
	}

	#formatTime(seconds) {
		if (!isFinite(seconds) || seconds < 0) return '0:00';
		// Always show hours if the total duration is >= 1 hour,
		// so the display width stays stable (e.g. "0:05:30" not "5:30").
		const forceHours = this.videoTarget.duration >= 3600;
		const hrs = Math.floor(seconds / 3600);
		const mins = Math.floor((seconds % 3600) / 60);
		const secs = Math.floor(seconds % 60);
		if (hrs > 0 || forceHours) {
			return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
		}
		return `${mins}:${secs.toString().padStart(2, '0')}`;
	}

	get #doShowControls() {
		if (this.#harlowMode) {
			return false;
		}
		// Show controls if recentInteraction, loading, paused,
		// button active, or recent touch seek, otherwise hide.
		return this.#recentInteraction ||
			this.loadingValue ||
			this.videoTarget.paused ||
			//controlsElement.pressed ||
			//controlsElement.hover ||
			this.#recentTouchSeek;
	}
}
