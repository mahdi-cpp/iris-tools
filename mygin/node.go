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

func (n *node) add(path string, handlers HandlersChain, fullPath string) {
	// نرمال‌سازی مسیر: حذف اسلش انتهایی به جز برای ریشه
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	n.addRecursive(path, handlers, fullPath)
}

func (n *node) addRecursive(path string, handlers HandlersChain, fullPath string) {
	// اگر مسیر خالی باشد، هندلرها را تنظیم کن
	if path == "" {
		n.handlers = handlers
		n.fullPath = fullPath
		return
	}

	// پیدا کردن طول پیشوند مشترک
	i := 0
	maxLen := min(len(n.path), len(path))
	for i < maxLen && n.path[i] == path[i] {
		i++
	}

	// اگر پیشوند مشترک کمتر از طول مسیر گره فعلی باشد، گره را تقسیم کن
	if i < len(n.path) {
		// تقسیم گره
		child := &node{
			path:      n.path[i:],
			children:  n.children,
			handlers:  n.handlers,
			fullPath:  n.fullPath,
			isParam:   n.isParam,
			paramName: n.paramName,
		}

		n.path = n.path[:i]
		n.children = []*node{child}
		n.handlers = nil
		n.fullPath = ""
		n.isParam = false
		n.paramName = ""
	}

	// مسیر باقی‌مانده
	remainingPath := path[i:]

	// اگر مسیر باقی‌مانده خالی باشد، هندلرها را تنظیم کن
	if remainingPath == "" {
		n.handlers = handlers
		n.fullPath = fullPath
		return
	}

	// بررسی برای پارامتر
	if remainingPath[0] == ':' {
		// پیدا کردن نام پارامتر
		end := 1
		for end < len(remainingPath) && remainingPath[end] != '/' {
			end++
		}

		paramName := remainingPath[1:end]
		remainingAfterParam := remainingPath[end:]

		// بررسی آیا گره پارامتری با همین نام وجود دارد
		for _, child := range n.children {
			if child.isParam && child.paramName == paramName {
				child.addRecursive(remainingAfterParam, handlers, fullPath)
				return
			}
		}

		// ایجاد گره پارامتری جدید
		paramNode := &node{
			path:      remainingPath[:end],
			isParam:   true,
			paramName: paramName,
		}

		n.children = append(n.children, paramNode)
		paramNode.addRecursive(remainingAfterParam, handlers, fullPath)
		return
	}

	// برای مسیرهای ثابت، فرزند موجود را پیدا کن یا ایجاد کن
	for _, child := range n.children {
		if !child.isParam && child.path != "" && child.path[0] == remainingPath[0] {
			child.addRecursive(remainingPath, handlers, fullPath)
			return
		}
	}

	// ایجاد گره جدید
	newNode := &node{
		path: remainingPath,
	}
	n.children = append(n.children, newNode)
	newNode.addRecursive("", handlers, fullPath)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// find attempts to find a matching route in the tree.
func (n *node) find(path string) (HandlersChain, map[string]string) {
	return n.findRecursive(path, make(map[string]string))
}

func (n *node) findRecursive(path string, params map[string]string) (HandlersChain, map[string]string) {
	// اگر مسیر جاری با پیشوند مسیر هدف منطبق باشد
	if len(path) >= len(n.path) && path[:len(n.path)] == n.path {
		remainingPath := path[len(n.path):]

		// اگر مسیر دقیقاً تمام شده باشد
		if remainingPath == "" {
			if n.handlers != nil {
				return n.handlers, params
			}
			return nil, nil
		}

		// ابتدا فرزندان ثابت را بررسی کن
		for _, child := range n.children {
			if !child.isParam {
				if handlers, foundParams := child.findRecursive(remainingPath, cloneParams(params)); handlers != nil {
					return handlers, foundParams
				}
			}
		}

		// سپس فرزندان پارامتری را بررسی کن
		for _, child := range n.children {
			if child.isParam {
				// پیدا کردن انتهای بخش پارامتر
				end := 0
				for end < len(remainingPath) && remainingPath[end] != '/' {
					end++
				}

				if end > 0 {
					paramValue := remainingPath[:end]
					newParams := cloneParams(params)
					newParams[child.paramName] = paramValue

					// اگر پارامتر تمام مسیر باقی‌مانده را پوشش دهد
					if end == len(remainingPath) {
						if child.handlers != nil {
							return child.handlers, newParams
						}
					} else {
						// اگر مسیر بیشتری باقی مانده، در فرزندان جستجو کن
						nextPath := remainingPath[end:]
						for _, grandChild := range child.children {
							if handlers, foundParams := grandChild.findRecursive(nextPath, newParams); handlers != nil {
								return handlers, foundParams
							}
						}
					}
				}
			}
		}
	}

	return nil, nil
}

func cloneParams(params map[string]string) map[string]string {
	newParams := make(map[string]string)
	for k, v := range params {
		newParams[k] = v
	}
	return newParams
}
