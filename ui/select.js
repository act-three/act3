import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	static targets = ["trigger", "content", "item", "label"];
	static values = { current: String };

	#outsideClick;

	connect() {
		this.#outsideClick = (ev) => {
			if (!this.element.contains(ev.target)) {
				this.close();
			}
		};
		document.addEventListener("click", this.#outsideClick);
		this.#syncFromValue();
	}

	disconnect() {
		document.removeEventListener("click", this.#outsideClick);
	}

	open() {
		if (this.element.hasAttribute("data-open")) return;
		this.element.setAttribute("data-open", "");
		this.#positionContent();
	}

	close() {
		this.element.removeAttribute("data-open");
		this.contentTarget.style.top = "";
		this.contentTarget.style.left = "";
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
				this.open();
				break;
			case "Escape":
				ev.preventDefault();
				this.close();
				this.triggerTarget.focus();
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
		const selected =
			content.querySelector("[data-selected]") ||
			this.itemTargets[0];
		if (!selected) return;

		const parentRect = this.element.getBoundingClientRect();
		const triggerRect = trigger.getBoundingClientRect();
		const contentRect = content.getBoundingClientRect();
		const itemRect = selected.getBoundingClientRect();

		// Vertical: align selected item center with trigger center.
		const triggerMid = triggerRect.top + triggerRect.height / 2;
		const itemMidInContent =
			itemRect.top - contentRect.top + itemRect.height / 2;
		let top = triggerMid - itemMidInContent - parentRect.top;

		// Horizontal: align item text with label text.
		let left = 0;
		if (this.hasLabelTarget) {
			const labelLeft = this.labelTarget.getBoundingClientRect().left;
			const itemPadLeft = parseFloat(
				getComputedStyle(selected).paddingLeft,
			);
			const itemTextLeft = itemRect.left + itemPadLeft;
			left = labelLeft - itemTextLeft;
		}

		// Clamp to viewport edges with 4px margin.
		const vw = window.innerWidth;
		const vh = window.innerHeight;
		const absTop = parentRect.top + top;
		if (absTop < 4) {
			top += 4 - absTop;
		} else if (absTop + contentRect.height > vh - 4) {
			top -= absTop + contentRect.height - (vh - 4);
		}
		const absLeft = parentRect.left + left;
		if (absLeft < 4) {
			left += 4 - absLeft;
		} else if (absLeft + contentRect.width > vw - 4) {
			left -= absLeft + contentRect.width - (vw - 4);
		}

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
