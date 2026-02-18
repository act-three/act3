import "./turbo.es2017-esm.js";
import { Application, Controller } from "./stimulus.js";
import Plyr from "./plyr.js";
window.Stimulus = Application.start();

Stimulus.register("dialog", class extends Controller {
	dismiss() {
		this.element.classList.add("hidden");
	}
});

const playerControls = `
<div class="plyr__controls-top absolute top-0 left-0 right-0 flex flex-row items-center gap-4 p-2.5 z-3">
    <button type="button" class="plyr__control" data-action="click->player#dismiss">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
    </button>
    <div class="plyr__title">{title}</div>
</div>
<div class="plyr__controls">
    <button type="button" class="plyr__controls__item plyr__control" data-plyr="play" aria-label="Play">
        <svg class="icon--pressed" role="presentation" focusable="false"><use xlink:href="{iconUrl}#plyr-pause"></use></svg>
        <svg class="icon--not-pressed" role="presentation" focusable="false"><use xlink:href="{iconUrl}#plyr-play"></use></svg>
        <span class="label--pressed plyr__tooltip" role="tooltip">Pause</span>
        <span class="label--not-pressed plyr__tooltip" role="tooltip">Play</span>
    </button>
    <div class="plyr__controls__item plyr__progress__container">
        <div class="plyr__progress">
            <input data-plyr="seek" type="range" min="0" max="100" step="0.01" value="0" autocomplete="off" role="slider" aria-label="Seek" aria-valuemin="0" aria-valuemax="100" aria-valuenow="0" id="plyr-seek-{id}">
            <progress class="plyr__progress__buffer" min="0" max="100" value="0" role="progressbar" aria-hidden="true">% buffered</progress>
            <span class="plyr__tooltip">00:00</span>
        </div>
    </div>
    <div class="plyr__controls__item plyr__time plyr__time--current" aria-label="Current time">00:00</div>
    <div class="plyr__controls__item plyr__time plyr__time--duration" aria-label="Duration">00:00</div>
    <div class="plyr__controls__item plyr__volume">
        <button type="button" class="plyr__control" data-plyr="mute">
            <svg class="icon--pressed" role="presentation" focusable="false"><use xlink:href="{iconUrl}#plyr-muted"></use></svg>
            <svg class="icon--not-pressed" role="presentation" focusable="false"><use xlink:href="{iconUrl}#plyr-volume"></use></svg>
            <span class="label--pressed plyr__tooltip" role="tooltip">Unmute</span>
            <span class="label--not-pressed plyr__tooltip" role="tooltip">Mute</span>
        </button>
        <input data-plyr="volume" type="range" min="0" max="1" step="0.05" value="1" autocomplete="off" role="slider" aria-label="Volume" aria-valuemin="0" aria-valuemax="100" aria-valuenow="100" id="plyr-volume-{id}">
    </div>
    <button type="button" class="plyr__controls__item plyr__control" data-plyr="fullscreen">
        <svg class="icon--pressed" role="presentation" focusable="false"><use xlink:href="{iconUrl}#plyr-exit-fullscreen"></use></svg>
        <svg class="icon--not-pressed" role="presentation" focusable="false"><use xlink:href="{iconUrl}#plyr-enter-fullscreen"></use></svg>
        <span class="label--pressed plyr__tooltip" role="tooltip">Exit fullscreen</span>
        <span class="label--not-pressed plyr__tooltip" role="tooltip">Enter fullscreen</span>
    </button>
</div>
`;

Stimulus.register("player", class extends Controller {
	static targets = ["video"];
	static values = { iconUrl: String, title: String };

	connect() {
		const controls = playerControls
			.replaceAll('{iconUrl}', this.iconUrlValue)
			.replaceAll('{title}', this.titleValue);
		this.player = new Plyr(this.videoTarget, {
			controls,
			iconUrl: this.iconUrlValue,
			loadSprite: false,
		});
		setTimeout(() => { this.player.play() }, 2000);
	}

	disconnect() {
		if (this.player) {
			this.player.destroy();
			this.player = null;
		}
	}

	dismiss() {
		this.element.remove();
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
