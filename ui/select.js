import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static targets = ["trigger", "content", "item", "label"];
	static values = { current: String };

	connect() {
		const content = this.contentTarget;
		if (!content.id) {
			content.id = "sel-" + Math.random().toString(36).slice(2, 10);
		}
		this.triggerTarget.setAttribute("popovertarget", content.id);
		this.#syncFromValue();
	}

	toggled(ev) {
		if (ev.newState === "open") {
			this.#positionContent();
		} else {
			this.triggerTarget.focus();
		}
	}

	close() {
		this.contentTarget.hidePopover();
	}

	selectItem(ev) {
		this.currentValue = ev.params.value;
		this.close();
		this.element.dispatchEvent(
			new CustomEvent("select:change", {
				detail: { value: this.currentValue },
				bubbles: true,
			}),
		);
	}

	currentValueChanged() {
		this.#syncFromValue();
	}

	keydown(ev) {
		switch (ev.key) {
			case "Enter":
			case " ":
				ev.preventDefault();
				this.contentTarget.showPopover();
				break;
			case "ArrowDown":
				ev.preventDefault();
				this.#focusItem(1);
				break;
			case "ArrowUp":
				ev.preventDefault();
				this.#focusItem(-1);
				break;
		}
	}

	// Position the content so the selected item's text aligns
	// with the trigger's label text, both vertically and
	// horizontally.
	#positionContent() {
		const content = this.contentTarget;
		const trigger = this.triggerTarget;
		const selected = content.querySelector("[data-selected]")
			|| this.itemTargets[0];
		if (!selected) return;

		// Reset so we can measure natural size.
		content.style.position = "fixed";
		content.style.margin = "0";
		content.style.top = "0";
		content.style.left = "0";
		content.style.minWidth = trigger.offsetWidth + "px";

		const triggerRect = trigger.getBoundingClientRect();
		const contentRect = content.getBoundingClientRect();
		const itemRect = selected.getBoundingClientRect();

		// Vertical: align selected item center with trigger center.
		const triggerMid = triggerRect.top + triggerRect.height / 2;
		const itemMidInContent = itemRect.top - contentRect.top + itemRect.height / 2;
		let top = triggerMid - itemMidInContent;

		// Horizontal: align item text with label text.
		let left = triggerRect.left;
		if (this.hasLabelTarget) {
			const labelLeft = this.labelTarget.getBoundingClientRect().left;
			const itemPadLeft = parseFloat(
				getComputedStyle(selected).paddingLeft,
			);
			const itemTextLeft = itemRect.left - contentRect.left + itemPadLeft;
			left = labelLeft - itemTextLeft;
		}

		// Clamp to viewport edges with 4px margin.
		const vw = window.innerWidth;
		const vh = window.innerHeight;
		const margin = 4;
		top = Math.max(margin, Math.min(top, vh - contentRect.height - margin));
		left = Math.max(margin, Math.min(left, vw - contentRect.width - margin));

		content.style.top = top + "px";
		content.style.left = left + "px";
	}

	#syncFromValue() {
		const val = this.currentValue;
		for (const t of this.itemTargets) {
			if (t.getAttribute("data-select-value-param") === val) {
				t.setAttribute("data-selected", "");
				if (this.hasLabelTarget) {
					this.labelTarget.textContent = t.textContent.trim();
				}
			} else {
				t.removeAttribute("data-selected");
			}
		}
	}

	#focusItem(dir) {
		const items = this.itemTargets;
		if (items.length === 0) return;
		const current = document.activeElement;
		const idx = items.indexOf(current);
		let next = idx + dir;
		if (next < 0) next = items.length - 1;
		if (next >= items.length) next = 0;
		items[next].focus();
	}
}
