package controller

import (
	"fmt"
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
		assertStringEquals(t, expectedExposer, config.Exposer, "Exposer")
		assertStringEquals(t, expectedDomain, config.Domain, "Domain")
/*
		assert.Equal(t, expectedExposer, config.Exposer, "Exposer")
		assert.Equal(t, expectedDomain, config.Domain, "Domain")
*/
		fmt.Printf("Config is %#v\n", config)
	}
}

func assertStringEquals(t *testing.T, expected, actual, message string) {
	if expected != actual {
		t.Errorf("%s was not equal. Expected %s but got %s\n", message, expected, actual)
	}
}
