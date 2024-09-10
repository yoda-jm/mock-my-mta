package smtp

type Configuration struct {
	Addr   string              `json:"addr"`
	Relays RelayConfigurations `json:"relays"`
}

type RelayConfigurations map[string]RelayConfiguration

func (r RelayConfigurations) Names() []string {
	names := make([]string, 0, len(r))
	for name, relay := range r {
		if !relay.Enabled {
			continue
		}
		names = append(names, name)
	}
	return names
}

func (r RelayConfigurations) Get(name string) (RelayConfiguration, bool) {
	relay, found := r[name]
	if !found || !relay.Enabled {
		return RelayConfiguration{}, false
	}
	return relay, found
}

type RelayConfiguration struct {
	Enabled   bool          `json:"enabled"`
	AutoRelay bool          `json:"auto_relay"`
	Addr      string        `json:"addr"`
	Username  string        `json:"username"`
	Password  string        `json:"password"`
	Mechanism RelayAuthMode `json:"mechanism"`
}

type RelayAuthMode string

const (
	RelayAuthModeNone    RelayAuthMode = "NONE"
	RelayAuthModePlain   RelayAuthMode = "PLAIN"
	RelayAuthModeLogin   RelayAuthMode = "LOGIN"
	RelayAuthModeCramMD5 RelayAuthMode = "CRAM-MD5"
)
