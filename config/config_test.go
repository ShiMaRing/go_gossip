package config

import "testing"

func TestLoadConfig(t *testing.T) {
	// Test case for LoadConfig
	_, err := LoadConfig("../resources/config_1.ini")
	if err != nil {
		t.Errorf("Failed to load config: %v", err)
	}
	//print all the config
	t.Logf("Config: %v", P2PConfig)
}
