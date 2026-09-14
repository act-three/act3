// Importing this module has no side effects. Call run to initialize Hi.

// The displayed nodes belong to the document, including across Domi's
// navigation snapshots and resyncs. This script changes only the opaque port.
const notes = [];
const delivered = new Set();
let port;

function present(node) {
	notes.push(node);
	if (port?.isConnected) port.appendChild(node);
}

export function run() {
	if (customElements.get("hi-note-port")) return;

	customElements.define(
		"hi-note-port",
		class extends HTMLElement {
			connectedCallback() {
				port = this;
				// A restored snapshot may contain old clones of displayed notes.
				// The document's original nodes are the authoritative copy.
				this.replaceChildren(...notes);
			}
		},
	);

	customElements.define(
		"hi-note-outbox",
		class extends HTMLElement {
			#observer = new MutationObserver(records => {
				for (const record of records) {
					for (const entry of record.addedNodes) this.#receive(entry);
				}
			});

			connectedCallback() {
				this.#observer.observe(this, { childList: true });
				for (const entry of this.children) this.#receive(entry);
			}

			disconnectedCallback() {
				this.#observer.disconnect();
			}

			#receive(entry) {
				const id = entry.getAttribute?.("domi-key");
				if (!id || delivered.has(id)) return;
				delivered.add(id);
				present(entry.cloneNode(true));
			}
		},
	);
}
