// Custom Turbo Stream action that assigns attributes from the
// template element onto each target, without removing existing
// attributes or touching children.
window.Turbo.StreamActions.set = function() {
	const source = this.templateContent.firstElementChild;
	if (!source) return;
	for (const target of this.targetElements) {
		for (const { name, value } of source.attributes) {
			target.setAttribute(name, value);
		}
	}
};

// Custom Turbo Stream action that replaces the browser URL
// if the current path matches the "from" attribute.
// Does not create a new history entry.
window.Turbo.StreamActions.url = function() {
	const from = this.getAttribute("from");
	const to = this.getAttribute("to");
	if (from && to && location.pathname === from) {
		history.replaceState(history.state, "", to);
	}
};
