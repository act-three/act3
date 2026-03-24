document.addEventListener("turbo:before-render", (event) => {
	// The wash image is chosen randomly on each render. When Turbo
	// shows a cached preview followed by a fresh response, the wash
	// changes twice in quick succession, causing a flicker. Preserve
	// the preview's wash so the fresh response doesn't swap it out.
	if (!document.documentElement.hasAttribute("data-turbo-preview")) return;
	const oldWash = document.querySelector(".v-media-wash");
	if (!oldWash) return;
	const newWash = event.detail.newBody.querySelector(".v-media-wash");
	if (!newWash) return;
	newWash.replaceWith(oldWash.cloneNode(true));
});
