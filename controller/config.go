package controller

import (
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/v2/yaml"
)

func LoadFile(path string) (*Config, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config file")
	}
	c, err := Load(string(content))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config file")
	}
	return c, nil
}

func Load(s string) (*Config, error) {
	cfg := &Config{}
	// If the entire config body is empty the UnmarshalYAML method is
	// never called. We thus have to set the DefaultConfig at the entry
	// point as well.
	*cfg = DefaultConfig
	err := yaml.Unmarshal([]byte(s), &cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config")
	}

	cfg.original = s

	return cfg, nil
}

type Config struct {
	Domain  string `yaml:"domain,omitempty"`
	Exposer string `yaml:"exposer"`

	// original is the input from which the config was parsed.
	original string
}

var (
	DefaultConfig = Config{}
)

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultConfig
	// We want to set c to the defaults and then overwrite it with the input.
	// To make unmarshal fill the plain data struct rather than calling UnmarshalYAML
	// again, we have to hide it using a type indirection.
	type plain Config
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}

	if len(c.Domain) == 0 {
		return fmt.Errorf("domain is required")
	}
	if len(c.Exposer) == 0 {
		return fmt.Errorf("exposer is required")
	}

	return nil
}

func (c Config) String() string {
	if c.original != "" {
		return c.original
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("<error creating config string: %s>", err)
	}
	return string(b)
}
