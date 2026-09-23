// Import the same digest-named module that ClientModule loaded.
const source = document.querySelector("script[src^=\"/-/domi/hi.\"]").src;
const Hi = await import(source);

document.addEventListener("click", event => {
	if (!event.target.closest("#client-note")) return;
	// This Hi button is handled locally, before Domi dispatches its message.
	event.stopPropagation();
	Hi.notify("client-note");
}, true);
