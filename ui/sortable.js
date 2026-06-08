import { Controller } from "../web/stimulus.js";
// web/sortable.js is a vendored copy of SortableJS 1.15.6 with a local
// patch: in _onTouchMove, when direction is "vertical", touchEvt.clientX
// is locked to the drag-start X so that hit-testing continues to work
// when the pointer drifts outside the container horizontally.
// forceFallback must be true for the patch to take effect (it only
// applies to the fallback/emulated drag path, not native drag events).
// Search for "LOCAL PATCH" in that file for the change.
import Sortable from "../web/sortable.js";

export default class extends Controller {
	#drag = null;

	connect() {
		this.sortable = Sortable.create(this.element, {
			group: "episodes",
			handle: ".u-sortable-handle",
			direction: "vertical",
			forceFallback: true,
			animation: 150,
			ghostClass: "u-sortable-ghost",
			dragClass: "u-sortable-drag",
			onStart: (e) => {
				this.#drag = { item: e.item, parent: e.from, next: e.item.nextSibling };
			},
			onEnd: (e) => this.#onEnd(e),
		});
	}

	disconnect() {
		this.sortable.destroy();
	}

	// Cancels a drag by reverting the element to its original position
	// before ending the drag, so SortableJS sees no change.
	cancel() {
		if (!this.#drag) return;
		this.#drag.parent.insertBefore(this.#drag.item, this.#drag.next);
		this.#drag = null;
		document.dispatchEvent(new PointerEvent("pointerup", { bubbles: true }));
	}

	#onEnd(e) {
		if (!this.#drag) return;
		const drag = this.#drag;
		this.#drag = null;

		// The row the drop landed before, in its dropped position, or
		// null if it dropped last. Captured before the revert below: the
		// revert moves only e.item, so this reference stays valid — and
		// in the destination container — in the clean DOM.
		const anchor = e.item.nextElementSibling;

		// Restore the pre-drag DOM so domi sees a clean tree, then hand
		// it the drop as an optimistic mutation. domi applies the move
		// synchronously (no paint between the revert and the re-apply,
		// so no bounce), rebases onto a derived version so stale frames
		// drop, and reports it; the server replays the move to
		// reconstruct what we show and diffs forward — an empty patch
		// when it agrees, a correction when it refuses. The event also
		// carries the fields onEpisodeMove reads to perform the move.
		drag.parent.insertBefore(drag.item, drag.next);

		const move = anchor
			? { op: "move", node: e.item, before: anchor }
			: { op: "move", node: e.item, into: e.to };
		const detail = {
			episodeId: e.item.dataset.episodeId,
			fromSeasonId: e.from.dataset.seasonId,
			seasonId: e.to.dataset.seasonId,
			index: e.newIndex,
			domi: { mutations: [move] },
		};
		this.element.dispatchEvent(
			new CustomEvent("change", { bubbles: true, detail }),
		);
	}
}
