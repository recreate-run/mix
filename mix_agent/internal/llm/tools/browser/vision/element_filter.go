package vision

// FilterInteractiveElements filters accessibility nodes to interactive elements within viewport
func FilterInteractiveElements(nodes []RawAccessibilityNode, viewport ViewportBounds) []Element {
	elements := make([]Element, 0, len(nodes))
	index := 0

	viewportBox := BoundingBox(viewport)

	for _, node := range nodes {
		// Check if interactive role
		if !isInteractiveRole(node.Role) {
			continue
		}

		// Skip zero-size elements
		if node.Bounds.Width == 0 || node.Bounds.Height == 0 {
			continue
		}

		// Check viewport intersection
		if !isInViewport(node.Bounds, viewportBox) {
			continue
		}

		// Add with sequential index
		elements = append(elements, Element{
			Index:     index,
			Role:      node.Role,
			Name:      node.Name,
			Bounds:    node.Bounds,
			BackendID: node.BackendID,
		})
		index++
	}

	return elements
}

// isInteractiveRole checks if a role is considered interactive
func isInteractiveRole(role string) bool {
	return interactiveRoles[role]
}

// isInViewport checks if an element intersects with the viewport
func isInViewport(elem, viewport BoundingBox) bool {
	return !(elem.X+elem.Width < viewport.X ||
		elem.X > viewport.X+viewport.Width ||
		elem.Y+elem.Height < viewport.Y ||
		elem.Y > viewport.Y+viewport.Height)
}
