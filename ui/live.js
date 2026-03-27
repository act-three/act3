// matchAddr checks whether an element's data-addr0, data-addr1, ...
// attributes match the given addr array exactly.
export function matchAddr(el, addr) {
	for (let i = 0; i < addr.length; i++) {
		if (el.getAttribute("data-addr" + i) !== addr[i]) return false;
	}
	// Ensure there are no extra addr attributes on the element.
	return !el.hasAttribute("data-addr" + addr.length);
}
