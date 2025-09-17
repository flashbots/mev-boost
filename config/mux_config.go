package config

import (
	"fmt"
	"io"
	"os"

	"github.com/flashbots/mev-boost/server/types"
	"gopkg.in/yaml.v3"
)

type MuxConfig struct {
	Policies []Policy  `yaml:"policies"`
	Mappings []Mapping `yaml:"mappings"`
}

type Policy struct {
	Name     string    `yaml:"name"`
	Relayers []Relayer `yaml:"relayers"`
}

type Relayer struct {
	Name       string            `yaml:"name"`
	URL        string            `yaml:"url"`
	HTTPHeader map[string]string `yaml:"http-header,omitempty"`
}

type Mapping struct {
	Name    string  `yaml:"name"`
	Policy  string  `yaml:"policy"`
	Filters Filters `yaml:"filters"`
}

type Filters struct {
	PublicKeys []string `yaml:"public_keys,omitempty"`
}

// LoadMuxConfig loads the muxing configuration from a yaml file
func LoadMuxConfig(configPath string) (*MuxConfig, error) {
	if configPath == "" {
		return nil, nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config MuxConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *MuxConfig) validate() error {
	if len(c.Policies) == 0 {
		return fmt.Errorf("atleast one policy must be defined")
	}

	policyNames := make(map[string]bool)
	// check if policies are valid
	for _, policy := range c.Policies {
		if policy.Name == "" {
			return fmt.Errorf("policy name cant be empty")
		}
		if policyNames[policy.Name] {
			return fmt.Errorf("duplicate policy name: %s", policy.Name)
		}
		policyNames[policy.Name] = true

		if len(policy.Relayers) == 0 {
			return fmt.Errorf("policy %s must have atleast one relayer", policy.Name)
		}

		// check for the relayers if valid
		for _, relayer := range policy.Relayers {
			if relayer.Name == "" {
				return fmt.Errorf("relayer name cant be empty in policy %s", policy.Name)
			}
			if relayer.URL == "" {
				return fmt.Errorf("relayer url cant be empty for %s in policy %s", relayer.Name, policy.Name)
			}
			if _, err := types.NewRelayEntry(relayer.URL); err != nil {
				return err
			}
		}
	}

	// check if mappings are valid
	// also check if they reference the correct policies
	for _, mapping := range c.Mappings {
		if mapping.Name == "" {
			return fmt.Errorf("mapping name cant be empty")
		}
		if mapping.Policy == "" {
			return fmt.Errorf("mapping %s must specify a policy", mapping.Name)
		}
		if !policyNames[mapping.Policy] {
			return fmt.Errorf("mapping %s references unknown policy: %s", mapping.Name, mapping.Policy)
		}
		if len(mapping.Filters.PublicKeys) == 0 {
			return fmt.Errorf("mapping %s must specify atleast one public key filter", mapping.Name)
		}
	}

	return nil
}

// GetPolicyForValidator returns the policy name for a given validator public key
// Returns empty string if no specific mapping is found (should use default behavior)
func (c *MuxConfig) GetPolicyForValidator(pubkey string) string {
	for _, mapping := range c.Mappings {
		for _, filterKey := range mapping.Filters.PublicKeys {
			if filterKey == pubkey {
				return mapping.Policy
			}
		}
	}
	return ""
}

func (c *MuxConfig) GetRelaysForPolicy(policyName string) ([]types.RelayEntry, error) {
	for _, policy := range c.Policies {
		if policy.Name == policyName {
			relays := make([]types.RelayEntry, 0, len(policy.Relayers))
			for _, relayer := range policy.Relayers {
				relay, err := types.NewRelayEntry(relayer.URL)
				if err != nil {
					return nil, fmt.Errorf("failed to create relay entry for %s: %w", relayer.URL, err)
				}
				relays = append(relays, relay)
			}
			return relays, nil
		}
	}
	return nil, fmt.Errorf("policy not found: %s", policyName)
}

func (c *MuxConfig) GetAllPolicies() []string {
	policies := make([]string, len(c.Policies))
	for _, policy := range c.Policies {
		policies = append(policies, policy.Name)
	}
	return policies
}
