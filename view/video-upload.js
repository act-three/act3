// video-upload.js manages a single video upload at module scope,
// surviving Turbo navigations. Progress is broadcast via custom
// events on document; any number of upload-progress controllers
// can connect and reflect the current state.
//
// The form's data-upload-target attribute identifies which item
// the upload is attached to (e.g. an episode or movie-edition ID),
// and is echoed back on every event so UI elements can show or
// hide based on whether they're bound to that same item.

import { notify } from "../ui/note-port.js";

let xhr = null;

// active returns the in-flight upload state, or null.
export function active() {
	if (!xhr) return null;
	return {
		progress: xhr._progress,
		filename: xhr._filename,
		target: xhr._target,
	};
}

// start begins a video upload from the given form element.
// The FormData is captured eagerly, so the form's DOM can be
// destroyed by Turbo navigation without affecting the upload.
export function start(form) {
	if (xhr) {
		notify("An upload is already in progress");
		return;
	}

	const fd = new FormData(form);
	const file = fd.get("video");
	const filename = file?.name || "video";
	const target = form.dataset.uploadTarget || "";

	const x = new XMLHttpRequest();
	x.open(form.method.toUpperCase(), form.action);
	x.setRequestHeader("Accept", "text/vnd.turbo-stream.html");
	x._progress = 0;
	x._filename = filename;
	x._target = target;
	xhr = x;

	document.documentElement.dataset.uploading = "";

	const onBeforeUnload = (e) => {
		e.preventDefault();
		e.returnValue = "";
	};
	window.addEventListener("beforeunload", onBeforeUnload);

	x.upload.addEventListener("progress", (e) => {
		if (!e.lengthComputable) return;
		x._progress = e.loaded / e.total;
		document.dispatchEvent(
			new CustomEvent("upload:progress", {
				detail: { progress: x._progress, filename, target },
			}),
		);
	});
	x.addEventListener("load", () => {
		if (xhr.status >= 200 && xhr.status < 300) {
			return;
		}
		const ct = x.getResponseHeader("Content-Type") || "";
		if (ct.includes("turbo-stream") && x.responseText) {
			Turbo.renderStreamMessage(x.responseText);
		} else {
			notify("Upload failed");
		}
	});
	x.addEventListener("error", () => notify("Could not reach the server"));
	x.addEventListener("abort", () => notify("Upload aborted"));
	x.addEventListener("loadend", () => {
		xhr = null;
		delete document.documentElement.dataset.uploading;
		window.removeEventListener("beforeunload", onBeforeUnload);
		document.dispatchEvent(
			new CustomEvent("upload:end", { detail: { target } }),
		);
	});

	document.dispatchEvent(
		new CustomEvent("upload:start", {
			detail: { progress: 0, filename, target },
		}),
	);

	x.send(fd);
}
