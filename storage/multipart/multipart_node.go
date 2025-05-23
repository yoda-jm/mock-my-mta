package multipart

type multipartNode struct {
	headers         map[string][]string
	parts           []node
	RelatedStartCID string // Content-ID of the start part for multipart/related
}

// multipart node implements the node interface
var _ node = multipartNode{}

func (m multipartNode) getHeaders() map[string][]string {
	return m.headers
}

func (m multipartNode) walkLeaves(fn walkLeavesFunc) walkStatus {
	for _, part := range m.parts {
		if status := part.walkLeaves(fn); status == stopWalk {
			return stopWalk
		}
	}
	return continueWalk
}
