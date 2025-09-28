package mygin

// node represents a node in the Radix Tree (Trie).
type node struct {
	path      string
	children  []*node
	handlers  HandlersChain
	fullPath  string // Full path of the route (e.g., "/users/:id")
	isParam   bool   // True if the node is a parameter node (starts with ':')
	paramName string // Name of the parameter (e.g., "id")
}

// addRoute is a wrapper for the core add function.
func (n *node) addRoute(path string, handlers HandlersChain) {
	n.add(path, handlers, path)
}

// add adds a new route to the tree.
func (n *node) add(path string, handlers HandlersChain, fullPath string) {

	// 💡 رفع مشکل: اگر دیگر مسیری برای افزودن نمانده است، Handler را تنظیم و بازگردانید.
	if len(path) == 0 {
		n.handlers = handlers
		n.fullPath = fullPath
		return
	}

	// 1. Find common prefix length (i)
	var i int
	maxLen := len(n.path)
	if len(path) < maxLen {
		maxLen = len(path)
	}

	for i = 0; i < maxLen; i++ {
		if n.path[i] != path[i] {
			break
		}
	}

	if i == len(n.path) { // Case 1: Full match on node's path (e.g., node="/api", path="/api/users")

		if len(path) == len(n.path) { // Route path ends exactly here (e.g., node="/api", path="/api")
			n.handlers = handlers
			n.fullPath = fullPath
			return
		}

		childPath := path[len(n.path):] // Remaining path for the child (e.g., "/users")

		// Check for dynamic parameters ('*' or ':') - must be checked before looking at children
		if childPath[0] == ':' || childPath[0] == '*' {
			// If a param node already exists, panic or handle conflict (simplified: just add/overwrite if it's the only segment)
			if len(n.children) > 0 && n.children[0].isParam {
				// Simple router often only allows one param node as the first child
				n.children[0].add(childPath, handlers, fullPath)
				return
			}

			paramNode := &node{
				path:      childPath,
				handlers:  handlers,
				fullPath:  fullPath,
				isParam:   true,
				paramName: childPath[1:]}

			// Add as the first child to prioritize parameter matching
			n.children = append([]*node{paramNode}, n.children...)
			return
		}

		// Find a matching child by the first character of the remaining path
		for _, child := range n.children {
			if !child.isParam && len(childPath) > 0 && child.path[0] == childPath[0] {
				child.add(childPath, handlers, fullPath)
				return
			}
		}

		// No matching child, create a new one
		newNode := &node{path: childPath, handlers: handlers, fullPath: fullPath}
		n.children = append(n.children, newNode)
		return

	} else { // Case 2: Partial match (Split the existing node)
		// Create a new parent node
		oldNode := *n

		// Update the current node (n) to be the common prefix
		*n = node{
			path:     oldNode.path[:i],
			children: make([]*node, 0, 2),
		}

		// Update the old node's path
		oldNode.path = oldNode.path[i:]

		// Create the new node for the remaining path
		newNode := &node{path: path[i:], handlers: handlers, fullPath: fullPath}

		// Add the old node and the new node as children to the new parent (n)
		n.children = append(n.children, &oldNode, newNode)
	}
}

// find attempts to find a matching route in the tree.
func (n *node) find(path string) (HandlersChain, map[string]string) {
	if len(path) == 0 {
		if n.handlers != nil {
			return n.handlers, nil
		}
		return nil, nil
	}

	// 1. Check current node path match
	if len(path) >= len(n.path) && path[:len(n.path)] == n.path {
		path = path[len(n.path):]

		// Route found exactly at this node
		if len(path) == 0 {
			return n.handlers, nil
		}

		// 2. Search children
		for _, child := range n.children {
			if !child.isParam && len(path) > 0 && child.path[0] == path[0] {
				// Recursive search on static child
				return child.find(path)
			}
		}

		// 3. Search parameter children (if any)
		for _, child := range n.children {
			if child.isParam {
				// For simple routers, treat the rest of the path as the param value
				if len(path) > 0 {
					params := make(map[string]string)

					// Simplified: assume the entire remaining path is the parameter value
					params[child.paramName] = path
					return child.handlers, params
				}
			}
		}
	}

	return nil, nil // No match found
}
