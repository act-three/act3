import { Controller } from "../web/stimulus.js";
import Sortable from "../web/sortable.js";

export default class extends Controller {
	connect() {
		this.sortable = Sortable.create(this.element, {
			group: "episodes",
			handle: ".u-sortable-handle",
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
