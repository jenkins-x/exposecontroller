package controller

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMapToConfig(t *testing.T) {
	expectedExposer := "Ingress"
	expectedDomain := "35.233.48.48.nip.io"

	data := map[string]string{
		"domain":  expectedDomain,
		"exposer": expectedExposer,
		"tls":     "false",
	}
	config, err := MapToConfig(data)
	if err != nil {
		t.Errorf("Failed to create Config %s\n", err)
	} else if config == nil {
		t.Error("No Config created!\n", err)
	} else {
		assert.Equal(t, expectedExposer, config.Exposer, "Exposer")
		assert.Equal(t, expectedDomain, config.Domain, "Domain")

		fmt.Printf("Config is %#v\n", config)
	}
}
