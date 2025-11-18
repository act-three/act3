import "./turbo.es2017-esm.js";
import { Application, Controller } from "./stimulus.js";
window.Stimulus = Application.start();

Stimulus.register("dialog", class extends Controller {
	dismiss() {
		this.element.classList.add("hidden");
	}
});

Stimulus.register("list", class extends Controller {
	static targets = ["item"];
	static values = {
		prefix: String,
		target: String,
		selected: Array,
	};

	#selectedPrevValue;
	#anchorID = "";
	#rangeEndID = "";
	#urls = new Map();

	initialize() {
		this.#initSelected();
		this.#showSelected();
	}

	selectedValueChanged(cur, old) {
		this.#selectedPrevValue = old;
		this.#showSelected();
	}

	select(ev) {
		if (ev.metaKey) {
			this.#selectToggle(ev);
		} else if (ev.shiftKey) {
			this.#selectRange(ev);
		} else {
			this.#selectOne(ev);
		}
		this.#navigate();
	}

	render() {
		this.#initSelected();
	}

	#initSelected() {
		const url = document.location;
		if (url.pathname.indexOf(this.prefixValue) != 0) {
			this.selectedValue = [];
			return;
		}
		let id = url.pathname.substring(this.prefixValue.length);
		const n = id.lastIndexOf("-");
		if (n >= 0) {
			id = id.substring(n + 1);
		}
		this.selectedValue = [id];
	}

	#navigate() {
		if (this.selectedValue.length == 1) {
			const selectedID = this.selectedValue[0];
			for (const t of this.itemTargets) {
				const targetID = t.getAttribute("data-list-id-param");
				if (targetID == selectedID) {
					const url = t.getAttribute("data-list-url-param");
					Turbo.visit(url, { frame: this.targetValue })
				}
			}
		}
	}

	#showSelected() {
		const selected = new Set(this.selectedValue);
		for (const t of this.itemTargets) {
			const id = t.getAttribute("data-list-id-param");
			if (selected.has(id)) {
				t.setAttribute("data-selected", "");
			} else {
				t.removeAttribute("data-selected");
			}
		}
	}

	#selectOne(ev) {
		const id = ev.params.id;
		this.selectedValue = [id];
		this.#urls.set(id, ev.params.url);
		this.#anchorID = id;
		this.#rangeEndID = "";
	}

	#selectToggle(ev) {
		const id = ev.params.id;
		const selected = new Set(this.selectedValue);
		if (selected.has(id)) {
			selected.delete(id);
		} else {
			selected.add(id);
		}
		this.selectedValue = Array.from(selected);
		this.#urls.set(id, ev.params.url);
		this.#anchorID = id;
		this.#rangeEndID = "";
	}

	#selectRange(ev) {
		const endID = ev.params.id;
		const selected = new Set(this.selectedValue);
		if (this.#rangeEndID != "") {
			this.#selectRangeMod(selected, this.#rangeEndID, false);
		}
		this.#selectRangeMod(selected, endID, true);
		this.selectedValue = Array.from(selected);
		this.#rangeEndID = endID;
	}

	#selectRangeMod(selected, endID, state) {
		let mod = false;
		for (const t of this.itemTargets) {
			let atStart = false;
			const id = t.getAttribute("data-list-id-param");
			if (id == this.#anchorID && mod) {
				break;
			}
			if (id == endID && !mod) {
				mod = true;
				atStart = true;
			}

			if (mod) {
				if (state) {
					selected.add(id);
				} else {
					selected.delete(id);
				}
			}

			if (id == endID && mod && !atStart) {
				break;
			}
			if (id == this.#anchorID && !mod) {
				mod = true;
			}
		}
	}
});


Stimulus.register("sidebar", class extends Controller {
	static targets = ["link"];

	initialize() {
		this.#showSelected(document.location);
	}

	visit(ev) {
		this.#showSelected(new URL(ev.detail.url));
	}

	#showSelected(url) {
		const current = this.#containingPaths(url.pathname);
		for (const t of this.linkTargets) {
			const path = t.getAttribute("href");
			if (current.has(path)) {
				t.setAttribute("data-selected", "");
			} else {
				t.removeAttribute("data-selected");
			}
		}
	}

	#containingPaths(path) {
		let s = new Set();
		while (path != "") {
			s.add(path);
			path = this.#dirname(path);
		}
		return s
	}

	#dirname(path) {
		const n = path.lastIndexOf("/");
		if (n < 0) {
			return "";
		}
		return path.substring(0, n);
	}
});

Stimulus.register("add-torrent", class extends Controller {
	static targets = ["picker", "button"];

	open() {
		this.pickerTarget.click();
	}

	upload() {
		this.element.requestSubmit(this.buttonTarget);
	}

	reset() {
		this.element.reset();
	}
});
