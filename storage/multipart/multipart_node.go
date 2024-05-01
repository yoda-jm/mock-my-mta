package multipart

type multipartNode struct {
	headers map[string][]string
	parts   []node
}

func (m multipartNode) GetHeaders() map[string][]string {
	return m.headers
}

func (m multipartNode) WalfLeaves(fn WalkLeavesFunc) WalkStatus {
	for _, part := range m.parts {
		if part.WalfLeaves(fn) == StopWalk {
			return StopWalk
		}
	}
	return ContinueWalk
}
