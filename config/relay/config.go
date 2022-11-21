package relay

type Config struct {
	ProposerConfig ProposerConfig `json:"proposer_config"`
	DefaultConfig  Relay          `json:"default_config"`
}

type ProposerConfig map[string]Relay

type Relay struct {
	FeeRecipient string  `json:"fee_recipient"`
	Builder      Builder `json:"builder"`
}

type Builder struct {
	Enabled  bool     `json:"enabled"`
	Relays   []string `json:"relays"`
	GasLimit string   `json:"gas_limit"`
}
