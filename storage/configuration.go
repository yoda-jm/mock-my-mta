package storage

// StorageLayerConfiguration defines a storage backend and its operation scopes.
type StorageLayerConfiguration struct {
	Type       string            `json:"type"`
	Scope      []string          `json:"scope"`
	Parameters map[string]string `json:"parameters"`
}

// Operation scopes that can be assigned to a storage layer.
const (
	ScopeRead   = "read"   // GetEmailByID, GetBodyVersion, GetAttachments, GetAttachment
	ScopeSearch = "search" // SearchEmails, GetMailboxes
	ScopeWrite  = "write"  // DeleteEmailByID, DeleteAllEmails
	ScopeRaw    = "raw"    // GetRawEmail
	ScopeCache  = "cache"  // Receives writes (Set) but is volatile — rebuilds on restart
	ScopeAll    = "all"    // Expands to all scopes
)

// hasScope returns true if the configuration includes the given scope.
func (c StorageLayerConfiguration) hasScope(scope string) bool {
	for _, s := range c.Scope {
		if s == ScopeAll || s == scope {
			return true
		}
	}
	return false
}

// isWritable returns true if the layer should receive writes (Set/setWithID).
func (c StorageLayerConfiguration) isWritable() bool {
	return c.hasScope(ScopeWrite) || c.hasScope(ScopeCache) || c.hasScope(ScopeAll)
}
