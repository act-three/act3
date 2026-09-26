// Importing this module has no side effects. Call run(Domi) to initialize Hi.

// Records, rather than snapshot DOM, own the display lifecycle. Delivery
// history is separate: retiring a note must never make it eligible again.
const notes = [];
const delivered = new Set();
const templates = new Map();
const lifetime = 4000;
const gap = 14;
const interactive = "button, a, input, select, textarea, [contenteditable], [tabindex], [domi-msg-click]";
let display;
let clone;
let hovered = false, focused = false, touchExpanded = false, gesture;
let previousFocus, pointer;
let suppressClick = false;

// Display a fresh copy of a template delivered by Go's RegisterNote command.
// Registration must have reached the browser before calling notify.
export function notify(name) {
	const t = templates.get(name);
	if (!t) throw new Error(`hi: unknown note template ${JSON.stringify(name)}`);
	show(t.cloneNode(true));
	sync();
}

function engaged() {
	return hovered || focused || touchExpanded || !!gesture;
}

function sync() {
	focused = !!(notes.length && display?.isConnected && display.querySelector(":focus-visible"));
	if (!notes.length) hovered = touchExpanded = false;
	const paused = engaged() || document.hidden || !display?.isConnected;
	const now = performance.now();
	for (const note of notes) {
		if (paused && note.timer !== undefined) {
			clearTimeout(note.timer);
			note.remaining = Math.max(0, note.deadline - now);
			note.timer = undefined;
		} else if (!paused && note.timer === undefined) {
			note.deadline = now + note.remaining;
			note.timer = setTimeout(() => {
				note.remaining = Math.max(0, note.deadline - performance.now());
				note.timer = undefined;
				if (!note.remaining) retire(note);
				sync();
			}, Math.min(note.remaining, 2147483647));
		}
	}
	layout();
}

function receive(entry) {
	const id = entry.getAttribute?.("domi-key");
	if (!id || delivered.has(id) || !entry.isConnected || !entry.parentElement.matches("hi-note-outbox")) return;
	const node = clone(entry);
	delivered.add(id);
	if (node.hasAttribute("data-note-template")) {
		const name = node.getAttribute("data-note-template");
		node.removeAttribute("data-note-template");
		node.removeAttribute("domi-key");
		templates.set(name, node);
		return;
	}
	show(node);
}

function show(node) {
	const duration = Number(node.dataset.duration);
	node.dataset.state = "active";
	node.tabIndex = 0;
	const note = {
		node,
		height: 0,
		mounted: false,
		remaining: duration > 0 ? duration : lifetime,
		timer: undefined,
	};
	node.addEventListener("click", event => {
		const control = event.target.closest("a[href], [domi-msg-click], [data-dismiss]");
		if (!control || !node.contains(control) || control.closest(":disabled, [aria-disabled=true]")) return;
		// Keep the exiting node attached so Domi's delegated listener can
		// dispatch the original click, including URL navigation.
		retire(note, { focusNext: event.detail === 0 });
		sync();
	});
	node.addEventListener("pointerdown", event => startDrag(event, note));
	notes.push(note);
	display?.appendChild(node);
}

function retire(note, { swipe = false, focusNext = true } = {}) {
	const index = notes.indexOf(note);
	if (index < 0) return;
	const hadFocus = note.node.contains(document.activeElement);
	const style = getComputedStyle(note.node);
	const transform = style.transform;
	const height = style.height;
	const covered = note.node.dataset.covered === "true";
	notes.splice(index, 1);
	clearTimeout(note.timer);
	if (gesture?.note === note) cancelDrag();
	note.node.style.setProperty("--exit-transform", transform === "none" ? "translateY(0)" : transform);
	note.node.style.height = height;
	note.node.dataset.exit = swipe ? "swipe" : covered ? "covered" : "normal";
	note.node.dataset.state = "exiting";
	note.node.inert = true;
	// Transition events can be canceled, or absent under reduced motion.
	setTimeout(() => note.node.remove(), 200);
	if (hadFocus) {
		if (focusNext && notes.length) {
			// Reveal retained notes before moving focus.
			layout();
			focusNote(notes[Math.min(index, notes.length - 1)]);
		} else restoreFocus();
	}
}

function focusNote(note) {
	const previous = previousFocus;
	note.node.focus({ preventScroll: true });
	// Inert can blur the old note, causing focusin to replace the page target.
	previousFocus = previous;
}

function restoreFocus() {
	if (previousFocus?.isConnected) previousFocus.focus({ preventScroll: true });
	if (display?.contains(document.activeElement) || document.activeElement === document.body) {
		const body = document.body;
		const old = body.getAttribute("tabindex");
		body.tabIndex = -1;
		body.focus({ preventScroll: true });
		if (old === null) body.removeAttribute("tabindex");
		else body.setAttribute("tabindex", old);
	}
}

function layout() {
	if (!display?.isConnected) return;
	// Measure once, before layout assigns a controlled height. Retained
	// notes keep their natural heights when moved into a restored snapshot.
	for (const note of notes) {
		if (!note.mounted) note.height = parseFloat(getComputedStyle(note.node).height);
	}
	const expanded = engaged();
	const visibleCount = Math.min(3, notes.length);
	display.dataset.expanded = String(expanded);
	display.style.pointerEvents = notes.length ? "auto" : "none";
	const front = notes.at(-1)?.height || 0;
	const activeElement = document.activeElement;
	let moveFocus = false;
	for (let i = 0; i < notes.length; i++) {
		const note = notes[i];
		const height = expanded ? note.height : front;
		const visible = i >= notes.length - visibleCount;
		moveFocus ||= !visible && note.node.contains(activeElement);
		note.node.dataset.visible = String(visible);
		note.node.inert = !visible;
		note.node.style.height = `${height}px`;
		note.node.dataset.covered = String(!expanded && i < notes.length - 1);
	}
	// Commit newly inserted shells at their full-height entry position and
	// zero opacity before setting all their animated targets together.
	if (notes.some(note => !note.mounted)) display.getBoundingClientRect();
	let offset = 0;
	let visibleHeight = 0;
	for (let i = notes.length - 1; i >= 0; i--) {
		const note = notes[i];
		const depth = notes.length - 1 - i;
		note.mounted = true;
		note.node.dataset.mounted = "true";
		note.node.style.setProperty("--offset", `${offset}px`);
		note.node.style.setProperty("--scale", expanded ? "1" : String(1 - depth * 0.05));
		offset += expanded ? note.height + gap : gap;
		if (depth < visibleCount) visibleHeight = expanded ? offset - gap : front + depth * gap;
	}
	display.style.height = `${visibleHeight}px`;
	if (moveFocus) focusNote(notes[notes.length - visibleCount]);
}

function startDrag(event, note) {
	const control = event.target.closest(interactive);
	if (
		event.button !== 0 || !event.isPrimary || gesture
		|| (control !== note.node && note.node.contains(control))
	) return;
	// Reveal a covered note before allowing a swipe to capture its transform.
	if (note.node.dataset.covered === "true") {
		touchExpanded = true;
		sync();
		return;
	}
	const transform = getComputedStyle(note.node).transform;
	gesture = {
		note,
		id: event.pointerId,
		start: event.clientY,
		time: event.timeStamp,
		distance: 0,
		moved: false,
		touch: event.pointerType === "touch",
	};
	note.node.style.setProperty("--drag-transform", transform === "none" ? "translateY(0)" : transform);
	note.node.style.setProperty("--swipe", "0px");
	note.node.dataset.swiping = "true";
	note.node.setPointerCapture(event.pointerId);
	sync();
}

function moveDrag(event) {
	if (!gesture || event.pointerId !== gesture.id) return;
	const g = gesture;
	g.distance = event.clientY - g.start;
	g.moved ||= Math.abs(g.distance) > 4;
	const delta = g.distance >= 0 ? g.distance : g.distance / (1.5 + Math.abs(g.distance) / 20);
	g.note.node.style.setProperty("--swipe", `${delta}px`);
}

function endDrag(event) {
	if (!gesture || event.pointerId !== gesture.id) return;
	const canceled = event.type !== "pointerup";
	const g = gesture;
	if (!canceled) moveDrag(event);
	const elapsed = event.timeStamp - g.time;
	const flick = elapsed > 0 && g.distance / elapsed > 0.11;
	const dismiss = !canceled && g.distance > 0 && (g.distance >= 45 || flick);
	if (g.moved) {
		suppressClick = true;
		setTimeout(() => {
			suppressClick = false;
		}, 400);
	} else if (!canceled && g.touch) touchExpanded = true;
	if (dismiss) retire(g.note, { swipe: true });
	else cancelDrag();
	sync();
}

function cancelDrag() {
	if (!gesture) return;
	const { note, id } = gesture;
	gesture = undefined;
	delete note.node.dataset.swiping;
	if (note.node.hasPointerCapture(id)) note.node.releasePointerCapture(id);
}

function initDisplay(element) {
	const hover = event => {
		if (event.pointerType !== "mouse") return;
		hovered = event.type === "pointerenter";
		if (!hovered) pointer = undefined;
		sync();
	};
	element.addEventListener("pointerenter", hover);
	element.addEventListener("pointerleave", hover);
	element.addEventListener("focusin", event => {
		if (!element.contains(event.relatedTarget)) previousFocus = event.relatedTarget;
		sync();
	});
	element.addEventListener("focusout", () => queueMicrotask(sync));
	element.addEventListener("keydown", event => {
		// Keyboard input can reveal focus without moving it.
		queueMicrotask(sync);
		if (event.key !== "Escape" || !element.contains(document.activeElement)) return;
		event.preventDefault();
		restoreFocus();
		sync();
	});
	element.addEventListener("click", event => {
		if (suppressClick && event.detail !== 0) {
			event.preventDefault();
			event.stopImmediatePropagation();
		}
	}, true);
	element.addEventListener("pointermove", moveDrag);
	element.addEventListener("pointerup", event => {
		if (gesture) {
			endDrag(event);
			return;
		}
		const note = event.target.closest("hi-note");
		if (event.pointerType === "touch" && note && event.target.closest(interactive) === note) {
			touchExpanded = true;
			sync();
		}
	});
	element.addEventListener("pointercancel", endDrag);
	element.addEventListener("lostpointercapture", endDrag);
}

function bind(element) {
	cancelDrag();
	display = element;
	hovered = touchExpanded = false;
	// Old snapshots can contain notes which have since been dismissed.
	// A reconnected display may already be correct; reinsertion would
	// discard focus acquired before this deferred binding pass.
	const nodes = notes.map(note => note.node);
	if (display.children.length !== nodes.length || nodes.some((node, i) => display.children[i] !== node)) {
		display.replaceChildren(...nodes);
	}
	if (pointer) hovered = display.contains(document.elementFromPoint(pointer.x, pointer.y));
	sync();
}

function restorePresentationPlacement(e, entry) {
	const saved = entry.placement || JSON.parse(e.dataset.hiPlacement || "null");
	if (!saved) return;
	for (const [property, [value, priority, applied, appliedPriority]] of Object.entries(saved)) {
		if (
			e.style.getPropertyValue(property) === applied && e.style.getPropertyPriority(property) === appliedPriority
		) {
			e.style.setProperty(property, value, priority);
		}
	}
	entry.placement = undefined;
	delete e.dataset.hiPlacement;
}

function sizePresentation(e, entry) {
	restorePresentationPlacement(e, entry);
	const fill = e.dataset.hiFill;
	if (!fill) return;
	// WebKit accepts a fitting authored position before considering
	// most-height/width. Measure the same candidates ourselves at opening.
	// https://bugs.webkit.org/show_bug.cgi?id=317916
	const vertical = fill !== "horizontal", horizontal = fill !== "vertical";
	const names = getComputedStyle(e).positionTryFallbacks.split(",").map(s => s.trim());
	const rules = new Map();
	function collect(list) {
		for (const rule of list) {
			if (names.includes(rule.name) && rule.style) rules.set(rule.name, rule.style);
			if (rule.cssRules) collect(rule.cssRules);
		}
	}
	for (const sheet of document.styleSheets) {
		if (sheet.href) continue;
		try {
			collect(sheet.cssRules);
		} catch { /* An inaccessible sheet cannot supply Hi's rules. */ }
	}
	// The empty first candidate lets the cascade supply the authored
	// placement. Alternatives reuse the Go-generated anchor expressions.
	const candidates = [
		[],
		...names.filter(name => rules.has(name)).map(name => {
			const style = rules.get(name);
			return [
				"inset-block-start",
				"inset-block-end",
				"inset-inline-start",
				"inset-inline-end",
				"align-self",
				"justify-self",
			]
				.map(property => [property, style.getPropertyValue(property), style.getPropertyPriority(property)])
				.filter(([, value]) => value);
		}),
	];
	const properties = new Set(["position-try-fallbacks", "position-try-order"]);
	for (const candidate of candidates) for (const [property] of candidate) properties.add(property);
	if (vertical) properties.add("height");
	if (horizontal) properties.add("width");
	const saved = Object.fromEntries(
		[...properties].map(
			property => [property, [e.style.getPropertyValue(property), e.style.getPropertyPriority(property)]],
		),
	);
	function apply(candidate) {
		for (const [property, [value, priority]] of Object.entries(saved)) {
			e.style.setProperty(property, value, priority);
		}
		e.style.setProperty("position-try-fallbacks", "none");
		e.style.setProperty("position-try-order", "normal");
		for (const [property, value, priority] of candidate) e.style.setProperty(property, value, priority);
	}
	const width = document.documentElement.clientWidth, height = document.documentElement.clientHeight;
	let best = candidates[0], most = -Infinity;
	for (const candidate of candidates) {
		apply(candidate);
		const rect = e.getBoundingClientRect();
		const fits = rect.left >= -.5 && rect.right <= width + .5 && rect.top >= -.5 && rect.bottom <= height + .5;
		// A filling axis occupies its entire available interval. The rect
		// includes native anchor-scroll adjustment; computed insets do not.
		const space = vertical ? rect.height : rect.width;
		if (fits && space > most) {
			best = candidate;
			most = space;
		}
	}
	apply(best);
	const style = getComputedStyle(e), heightUsed = style.height, widthUsed = style.width;
	if (vertical) e.style.height = heightUsed;
	if (horizontal) e.style.width = widthUsed;
	// Explicit sizes also give Safari's descendant scroll views the actual
	// chosen viewport. Keep anchor expressions so scrolling still tracks
	// the trigger without resizing the open presentation.
	for (const [property, values] of Object.entries(saved)) {
		values.push(e.style.getPropertyValue(property), e.style.getPropertyPriority(property));
	}
	entry.placement = saved;
	// History clones must be able to discard these owned inline overrides.
	e.dataset.hiPlacement = JSON.stringify(saved);
}

function initPresentations() {
	const entries = new Map();
	const stack = [];
	let pressedOutside, clickedOutside, pending = false;
	const top = () => stack.at(-1);
	const isDialog = e => e.dataset.hiPresentation === "dialog";
	const shown = e => e.matches(isDialog(e) ? ":modal" : ":popover-open");
	const registered = e => e.isConnected && entries.get(e).proxy.parentElement === e;
	const requested = e => registered(e) && e.dataset.hiOpen === "true";
	const dismiss = e => {
		// Domi delegates standard events. This inert proxy carries the
		// application's message without treating content clicks as dismissals.
		entries.get(e).proxy.click();
	};
	function hide(e) {
		if (isDialog(e)) {
			if (e.open) e.close();
		} else if (shown(e)) e.hidePopover();
	}
	function close(e) {
		const index = stack.indexOf(e);
		if (index >= 0) stack.splice(index, 1);
		const entry = entries.get(e);
		if (pressedOutside?.element === e) pressedOutside = undefined;
		if (clickedOutside === e) clickedOutside = undefined;
		// Removal can reset activeElement before the observer runs.
		const hadFocus = e.contains(document.activeElement)
			|| (!e.isConnected && entry.focusWithin && document.activeElement === document.body);
		hide(e);
		restorePresentationPlacement(e, entry);
		const previous = entry.focus?.deref();
		entry.focus = entry.active = entry.reconnectFocus = undefined;
		entry.focusWithin = false;
		if (hadFocus && previous?.isConnected) previous.focus({ preventScroll: true });
	}
	function schedule() {
		if (pending) return;
		pending = true;
		queueMicrotask(() => {
			pending = false;
			reconcile();
		});
	}
	function reconcile() {
		// Close in logical presentation order, independent of DOM order.
		for (const e of [...stack].reverse()) if (!requested(e)) close(e);
		const added = new Set();
		for (const [e, entry] of entries) {
			if (!registered(e)) {
				entry.observer.disconnect();
				entry.listeners.abort();
				entries.delete(e);
				continue;
			}
			if (!requested(e)) {
				close(e);
				continue;
			}
			if (!stack.includes(e)) {
				stack.push(e);
				added.add(e);
			}
		}
		const first = stack.findIndex(e => !shown(e));
		if (first >= 0) {
			let focus = document.activeElement;
			if (focus === document.body) {
				focus = [...stack].reverse().map(e => entries.get(e).reconnectFocus?.deref())
					.find(e => e?.isConnected);
			}
			// Moving a DOM subtree drops native top-layer membership. Restore
			// the affected suffix, including still-shown surfaces above it,
			// so neither DOM order nor repair order changes which is on top.
			for (let i = stack.length - 1; i >= first; i--) hide(stack[i]);
			for (let i = first; i < stack.length; i++) {
				const e = stack[i], entry = entries.get(e);
				if (added.has(e)) entry.focus = new WeakRef(document.activeElement);
				if (isDialog(e)) e.showModal();
				else e.showPopover();
				if (added.has(e)) sizePresentation(e, entry);
				entry.focusWithin = e.contains(document.activeElement);
			}
			if (!added.size && focus?.isConnected) focus.focus({ preventScroll: true });
		}
		for (const entry of entries.values()) entry.reconnectFocus = undefined;
	}
	let resizeFrame;
	window.addEventListener("resize", () => {
		cancelAnimationFrame(resizeFrame);
		resizeFrame = requestAnimationFrame(() => {
			for (const e of stack) if (shown(e) && entries.get(e).placement) sizePresentation(e, entries.get(e));
		});
	});
	document.addEventListener("focusin", event => {
		for (const [e, entry] of entries) {
			entry.focusWithin = e.contains(event.target);
			entry.active = entry.focusWithin ? new WeakRef(event.target) : undefined;
		}
	});
	document.addEventListener("focusout", () =>
		queueMicrotask(() => {
			for (const [e, entry] of entries) {
				if (e.isConnected) entry.focusWithin = e.contains(document.activeElement);
			}
		}));
	customElements.define(
		"hi-dismiss",
		class extends HTMLElement {
			#host;
			connectedCallback() {
				const e = this.#host = this.parentElement;
				if (!e?.matches("[data-hi-presentation]")) return;
				let entry = entries.get(e);
				if (!entry) {
					entry = { observer: new MutationObserver(schedule), listeners: new AbortController() };
					entries.set(e, entry);
					entry.observer.observe(e, { attributes: true, attributeFilter: ["data-hi-open", "open"] });
					const options = { signal: entry.listeners.signal };
					e.addEventListener("toggle", schedule, options);
					e.addEventListener("cancel", event => {
						if (!isDialog(e)) return;
						event.preventDefault();
						if (top() === e) dismiss(e);
					}, options);
				}
				entry.proxy = this;
				schedule();
			}
			disconnectedCallback() {
				const entry = entries.get(this.#host);
				if (!entry || entry.proxy !== this) return;
				entry.reconnectFocus = entry.focusWithin ? entry.active : undefined;
				schedule();
			}
		},
	);
	document.addEventListener("keydown", event => {
		const e = top();
		if (!e || event.key !== "Escape" || event.defaultPrevented || event.isComposing) return;
		// Prevent native dialog cancellation so precisely the topmost Hi
		// surface requests dismissal, even for a popover inside a dialog.
		event.preventDefault();
		if (!event.repeat) dismiss(e);
	});
	document.addEventListener("pointerdown", event => {
		const e = top();
		clickedOutside = undefined;
		pressedOutside = event.button === 0 && e && !isDialog(e) && !e.contains(event.target)
			? { element: e, id: event.pointerId }
			: undefined;
	}, true);
	document.addEventListener("pointerup", event => {
		const pressed = pressedOutside;
		pressedOutside = undefined;
		const e = pressed?.element;
		if (e && pressed.id === event.pointerId && e === top() && !e.contains(event.target)) clickedOutside = e;
	}, true);
	document.addEventListener("click", event => {
		const e = clickedOutside;
		clickedOutside = undefined;
		// Dispatch after the outside control's own action. In particular,
		// a toggle on the trigger must precede the idempotent close request.
		if (e && e === top() && !e.contains(event.target)) dismiss(e);
	});
	document.addEventListener("pointercancel", () => {
		pressedOutside = clickedOutside = undefined;
	}, true);
}

export function run(domi) {
	if (customElements.get("hi-note-display")) return;
	clone = domi.clone;
	initPresentations();
	document.addEventListener("visibilitychange", sync);
	document.addEventListener("pointermove", event => {
		if (event.pointerType === "mouse") pointer = { x: event.clientX, y: event.clientY };
	});
	document.addEventListener("pointerdown", event => {
		queueMicrotask(sync);
		suppressClick = false;
		if (display && !display.contains(event.target)) {
			touchExpanded = false;
			sync();
		}
	}, true);
	customElements.define(
		"hi-note-display",
		class extends HTMLElement {
			constructor() {
				super();
				initDisplay(this);
			}
			connectedCallback() {
				// Domi's handler version is available after the patch batch.
				queueMicrotask(() => {
					if (this.isConnected) bind(this);
				});
			}
			disconnectedCallback() {
				queueMicrotask(() => {
					if (display !== this || this.isConnected) return;
					cancelDrag();
					hovered = touchExpanded = false;
					sync();
				});
			}
		},
	);
	customElements.define(
		"hi-note-outbox",
		class extends HTMLElement {
			#observer = new MutationObserver(records => {
				for (const record of records) for (const entry of record.addedNodes) receive(entry);
				sync();
			});
			connectedCallback() {
				this.#observer.observe(this, { childList: true });
				queueMicrotask(() => {
					if (!this.isConnected) return;
					for (const entry of this.children) receive(entry);
					sync();
				});
			}
			disconnectedCallback() {
				this.#observer.disconnect();
			}
		},
	);
}
