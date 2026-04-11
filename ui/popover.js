import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static values = { trigger: String };

	connect() {
		const trigger = document.getElementById(this.triggerValue);
		this.trigger = trigger;
		const panel = this.element.querySelector(".u-popover-panel");
		if (!trigger || !panel) return;
		const anchor = trigger.getBoundingClientRect();
		const gap = 4;
		const pw = panel.offsetWidth, ph = panel.offsetHeight;
		let left = anchor.left + anchor.width / 2 - pw / 2;
		let top = anchor.bottom + gap;
		if (top + ph > window.innerHeight - 8 && anchor.top - gap - ph >= 8) {
			top = anchor.top - gap - ph;
		}
		left = Math.max(8, Math.min(left, window.innerWidth - pw - 8));
		top = Math.max(8, Math.min(top, window.innerHeight - ph - 8));
		panel.style.left = left + "px";
		panel.style.top = top + "px";
	}

	disconnect() {
		if (this.trigger) {
			this.trigger.style.visibility = "";
			this.trigger.removeAttribute("aria-expanded");
		}
	}

	close() {
		this.element.remove();
	}
}
