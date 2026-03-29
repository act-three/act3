import { Controller } from "../web/stimulus.js";

const GAP = 14;
const DURATION = 5000;
const VISIBLE = 3;
const SWIPE_THRESHOLD = 20;
const VELOCITY_THRESHOLD = 0.11;

// notify appends an error note to the note-port from JS.
// The note-port controller picks it up automatically via
// noteTargetConnected.
export function notify(msg, variant = "error") {
	const port = document.getElementById("note-port");
	if (!port) return;

	const title = document.createElement("div");
	title.className = "u-note-title";
	title.textContent = msg;

	const note = document.createElement("div");
	note.className = "u-note";
	note.setAttribute("role", "status");
	note.setAttribute("aria-live", "polite");
	note.setAttribute("data-variant", variant);
	note.setAttribute("data-note-port-target", "note");
	note.setAttribute(
		"data-action",
		"pointerdown->note-port#swipeStart "
			+ "pointermove->note-port#swipeMove "
			+ "pointerup->note-port#swipeEnd",
	);
	note.appendChild(title);

	port.appendChild(note);
}

export default class extends Controller {
	static targets = ["note"];

	togglePaused() {
		if (document.hidden) {
			this.#pauseAllTimers();
		} else {
			this.#resumeAllTimers();
		}
	}

	noteTargetConnected(el) {
		el.style.setProperty(
			"--initial-height",
			el.offsetHeight + "px",
		);

		requestAnimationFrame(() => {
			el.setAttribute("data-mounted", "");
			this.#layout();
		});

		this.#startTimer(el);

		for (const t of this.noteTargets.slice(0, -VISIBLE)) {
			this.dismiss(t);
		}
	}

	noteTargetDisconnected() {
		this.#layout();
	}

	pause() {
		this.#hovered = true;
		this.#pauseAllTimers();
		this.#layout();
	}

	resume() {
		this.#hovered = false;
		this.#resumeAllTimers();
		this.#layout();
	}

	dismiss(el) {
		this.#clearTimer(el);
		el.removeAttribute("data-swiping");
		el.style.removeProperty("--swipe");
		el.setAttribute("data-dismissed", "");
		el.removeAttribute("data-mounted");
		el.addEventListener("transitionend", () => {
			el.remove();
		}, { once: true });
		setTimeout(() => {
			if (el.parentNode) el.remove();
		}, 600);
	}

	// --- swipe to dismiss ---

	swipeStart(e) {
		if (e.target.closest("button, a")) {
			return;
		}
		const el = e.target.closest(
			"[data-note-port-target='note']",
		);
		if (!el) return;
		el.setPointerCapture(e.pointerId);
		this.#swipe = {
			el,
			startY: e.clientY,
			startTime: Date.now(),
		};
		el.setAttribute("data-swiping", "");
	}

	swipeMove(e) {
		if (!this.#swipe) return;
		const { el, startY } = this.#swipe;
		let dy = e.clientY - startY;

		// Friction when dragging upward (against dismiss).
		if (dy < 0) dy = dy * 0.2;

		el.style.setProperty("--swipe", dy + "px");
	}

	swipeEnd(e) {
		if (!this.#swipe) return;
		const { el, startY, startTime } = this.#swipe;
		this.#swipe = null;

		const dy = e.clientY - startY;
		const dt = Date.now() - startTime;
		const velocity = Math.abs(dy) / dt;

		if (
			dy > SWIPE_THRESHOLD
			|| velocity > VELOCITY_THRESHOLD
		) {
			// Animate out via swipe-out keyframe.
			this.#clearTimer(el);
			el.removeAttribute("data-swiping");
			el.setAttribute("data-swipe-out", "");
			el.removeAttribute("data-mounted");
			el.addEventListener("animationend", () => {
				el.remove();
			}, { once: true });
			setTimeout(() => {
				if (el.parentNode) el.remove();
			}, 300);
		} else {
			// Snap back.
			el.removeAttribute("data-swiping");
			el.style.setProperty("--swipe", "0px");
		}
	}

	// --- timers ---

	#hovered = false;
	#timers = new WeakMap();
	#swipe = null;

	#startTimer(el) {
		this.#clearTimer(el);
		if (this.#hovered || document.hidden) return;
		const id = setTimeout(
			() => this.dismiss(el),
			DURATION,
		);
		this.#timers.set(el, id);
	}

	#clearTimer(el) {
		const id = this.#timers.get(el);
		if (id) {
			clearTimeout(id);
			this.#timers.delete(el);
		}
	}

	#pauseAllTimers() {
		for (const el of this.noteTargets) {
			this.#clearTimer(el);
		}
	}

	#resumeAllTimers() {
		if (this.#hovered || document.hidden) return;
		for (const el of this.noteTargets) {
			this.#startTimer(el);
		}
	}

	// --- layout ---

	#layout() {
		const notes = this.noteTargets.filter(
			(n) =>
				!n.hasAttribute("data-dismissed")
				&& !n.hasAttribute("data-swipe-out"),
		);
		const count = notes.length;
		const expanded = this.#hovered;

		const frontHeight = count > 0
			? this.#height(notes[count - 1])
			: 0;

		let heightsBefore = 0;
		for (let i = count - 1; i >= 0; i--) {
			const note = notes[i];
			const idx = count - 1 - i;
			const h = this.#height(note);

			note.style.zIndex = count - idx;
			note.style.setProperty("--index", idx);
			note.style.setProperty(
				"--front-toast-height",
				frontHeight + "px",
			);
			if (idx === 0) {
				note.setAttribute("data-front", "");
			} else {
				note.removeAttribute("data-front");
			}

			if (expanded) {
				note.style.setProperty(
					"--offset",
					heightsBefore + "px",
				);
				heightsBefore += h + GAP;
			} else {
				note.style.setProperty(
					"--offset",
					(idx * GAP) + "px",
				);
			}
		}

		if (expanded) {
			this.element.style.setProperty(
				"--port-height",
				(heightsBefore + 24) + "px",
			);
		} else {
			this.element.style.setProperty(
				"--port-height",
				(frontHeight + VISIBLE * GAP + 24) + "px",
			);
		}
	}

	#height(el) {
		return parseInt(
			el.style.getPropertyValue("--initial-height"),
			10,
		) || el.offsetHeight;
	}
}
