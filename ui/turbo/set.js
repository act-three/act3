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
