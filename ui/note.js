import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static values = {
		duration: { type: Number, default: 5000 },
	};

	connect() {
		this.#remaining = this.durationValue;
		this.#startTimer();
	}

	disconnect() {
		this.#clearTimer();
	}

	dismiss() {
		this.#clearTimer();
		this.element.setAttribute("data-state", "closed");
		this.element.addEventListener("animationend", () => {
			this.element.remove();
		}, { once: true });
	}

	pause() {
		if (!this.#timerID) return;
		this.#remaining -= Date.now() - this.#started;
		this.#clearTimer();
	}

	resume() {
		if (this.#timerID) return;
		if (this.#remaining <= 0) return;
		this.#startTimer();
	}

	#timerID;
	#started;
	#remaining;

	#startTimer() {
		this.#started = Date.now();
		this.#timerID = setTimeout(() => this.dismiss(), this.#remaining);
	}

	#clearTimer() {
		if (this.#timerID) {
			clearTimeout(this.#timerID);
			this.#timerID = null;
		}
	}
}
