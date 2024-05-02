package multipart

type multipartNode struct {
	headers map[string][]string
	parts   []node
}

// multipart node implements the node interface
var _ node = multipartNode{}

func (m multipartNode) getHeaders() map[string][]string {
	return m.headers
}

func (m multipartNode) walfLeaves(fn walkLeavesFunc) walkStatus {
	for _, part := range m.parts {
		if part.walfLeaves(fn) == stopWalk {
			return stopWalk
		}
	}
	return continueWalk
}
