import { Controller } from "../web/stimulus.js";
import { notify } from "../ui/note-port.js";

// upload drives the image and video upload forms: the visible
// button opens the hidden file picker, and a chosen file is sent
// with fetch — a native form submission would navigate the page and
// tear down the domi connection. Upload progress and button state
// are rendered by the server from the model's upload registry; the
// only client-owned state is a beforeunload guard while uploads are
// in flight.

let inflight = 0;

function onBeforeUnload(e) {
	e.preventDefault();
	e.returnValue = "";
}

export default class extends Controller {
	static targets = ["picker"];

	open() {
		this.pickerTarget.click();
	}

	upload() {
		const form = this.element;
		// Copy the form's entries, inserting the file's exact size
		// just before the file itself so the server knows the total
		// when the bytes start arriving (the request Content-Length
		// includes multipart framing).
		const body = new FormData();
		for (const [name, value] of new FormData(form)) {
			if (value instanceof File) body.append("size", value.size);
			body.append(name, value);
		}
		form.reset();
		if (inflight++ === 0) {
			window.addEventListener("beforeunload", onBeforeUnload);
		}
		fetch(form.action, { method: "POST", body })
			.then(
				(res) => {
					if (!res.ok) notify("Upload failed");
				},
				() => notify("Could not reach the server"),
			)
			.finally(() => {
				if (--inflight === 0) {
					window.removeEventListener("beforeunload", onBeforeUnload);
				}
			});
	}
}
