import { Controller } from "../web/stimulus.js";

export default class extends Controller {
	filter(e) {
		const terms = e.target.value.toLowerCase().split(/\s+/).filter(Boolean);
		const searching = terms.length > 0;
		for (const item of this.element.querySelectorAll(".u-picker-item")) {
			const words = (item.dataset.filterText || "").toLowerCase().split(/\s+/);
			const match = terms.every(t => {
				const ep = t.match(/^e(\d+)$/);
				if (ep) {
					const n = parseInt(ep[1], 10);
					return words.some(w => {
						const m = w.match(/^s\d+e(\d+)$/);
						return m && parseInt(m[1], 10) === n;
					});
				}
				return words.some(w => w.startsWith(t));
			});
			item.toggleAttribute("data-search-hidden", searching && !match);
		}
	}
}
