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
	connect() {
		this.sortable = Sortable.create(this.element, {
			group: "episodes",
			handle: ".u-sortable-handle",
			direction: "vertical",
			forceFallback: true,
			animation: 150,
			ghostClass: "u-sortable-ghost",
			dragClass: "u-sortable-drag",
			onEnd: (e) => this.#onEnd(e),
		});
	}

	disconnect() {
		this.sortable.destroy();
	}

	#onEnd(e) {
		const body = new URLSearchParams();
		body.set("episode-id", e.item.dataset.episodeId);
		body.set("from-season-id", e.from.dataset.seasonId);
		body.set("season-id", e.to.dataset.seasonId);
		body.set("index", e.newIndex);

		fetch("/-/do/episode-move", { method: "POST", body });
	}
}
