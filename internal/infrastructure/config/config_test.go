// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package config

import (
	"flag"
	"os"
	"testing"
)

// saveFlags saves the current flag state
func saveFlags() *flag.FlagSet {
	return flag.CommandLine
}

// restoreFlags restores the flag state
func restoreFlags(saved *flag.FlagSet) {
	flag.CommandLine = saved
}

func TestConfig_NewInstance(t *testing.T) {
	config := &Config{
		Host:     "0.0.0.0",
		Port:     "8080",
		Debug:    false,
		JWKSUrl:  "http://heimdall:4457/.well-known/jwks",
		Audience: "lfx-v2-access-check",
		Issuer:   "heimdall",
		NATSUrl:  "nats://nats:4222",
	}

	if config == nil {
		t.Error("Config should not be nil")
	}
	if config.Host != "0.0.0.0" {
		t.Errorf("Expected Host to be '0.0.0.0', got '%s'", config.Host)
	}
	if config.Port != "8080" {
		t.Errorf("Expected Port to be '8080', got '%s'", config.Port)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Save original flags
	originalFlags := saveFlags()
	defer restoreFlags(originalFlags)

	// Clear environment variables
	clearEnvVars()
	defer clearEnvVars()

	// Create new flag set to avoid conflicts
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	config := LoadConfig()

	// Check default values
	if config.Host != "0.0.0.0" {
		t.Errorf("Expected default Host to be '0.0.0.0', got '%s'", config.Host)
	}
	if config.Port != "8080" {
		t.Errorf("Expected default Port to be '8080', got '%s'", config.Port)
	}
	if config.Debug != false {
		t.Errorf("Expected default Debug to be false, got %v", config.Debug)
	}
	if config.JWKSUrl != "http://heimdall:4457/.well-known/jwks" {
		t.Errorf("Expected default JWKSUrl, got '%s'", config.JWKSUrl)
	}
	if config.Audience != "lfx-v2-access-check" {
		t.Errorf("Expected default Audience to be 'access-check', got '%s'", config.Audience)
	}
	if config.Issuer != "heimdall" {
		t.Errorf("Expected default Issuer to be 'heimdall', got '%s'", config.Issuer)
	}
	if config.NATSUrl != "nats://nats:4222" {
		t.Errorf("Expected default NATSUrl to be 'nats://nats:4222', got '%s'", config.NATSUrl)
	}
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	// Save original flags
	originalFlags := saveFlags()
	defer restoreFlags(originalFlags)

	// Clear environment variables first
	clearEnvVars()
	defer clearEnvVars()

	// Create new flag set to avoid conflicts
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	// Set environment variables
	os.Setenv("HOST", "test-host")
	os.Setenv("PORT", "9090")
	os.Setenv("DEBUG", "true")
	os.Setenv("JWKS_URL", "http://test-jwks:8000/.well-known/jwks")
	os.Setenv("AUDIENCE", "test-audience")
	os.Setenv("ISSUER", "test-issuer")
	os.Setenv("NATS_URL", "nats://test-nats:4222")

	config := LoadConfig()

	// Check environment variable values are used
	if config.Host != "test-host" {
		t.Errorf("Expected Host from env to be 'test-host', got '%s'", config.Host)
	}
	if config.Port != "9090" {
		t.Errorf("Expected Port from env to be '9090', got '%s'", config.Port)
	}
	if config.Debug != true {
		t.Errorf("Expected Debug from env to be true, got %v", config.Debug)
	}
	if config.JWKSUrl != "http://test-jwks:8000/.well-known/jwks" {
		t.Errorf("Expected JWKSUrl from env, got '%s'", config.JWKSUrl)
	}
	if config.Audience != "test-audience" {
		t.Errorf("Expected Audience from env to be 'test-audience', got '%s'", config.Audience)
	}
	if config.Issuer != "test-issuer" {
		t.Errorf("Expected Issuer from env to be 'test-issuer', got '%s'", config.Issuer)
	}
	if config.NATSUrl != "nats://test-nats:4222" {
		t.Errorf("Expected NATSUrl from env to be 'nats://test-nats:4222', got '%s'", config.NATSUrl)
	}
}

func TestLoadConfig_PartialEnvironmentVariables(t *testing.T) {
	// Save original flags
	originalFlags := saveFlags()
	defer restoreFlags(originalFlags)

	// Clear environment variables first
	clearEnvVars()
	defer clearEnvVars()

	// Create new flag set to avoid conflicts
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	// Set only some environment variables
	os.Setenv("HOST", "partial-host")
	os.Setenv("DEBUG", "1")
	os.Setenv("AUDIENCE", "partial-audience")

	config := LoadConfig()

	// Check mix of env vars and defaults
	if config.Host != "partial-host" {
		t.Errorf("Expected Host from env to be 'partial-host', got '%s'", config.Host)
	}
	if config.Port != "8080" {
		t.Errorf("Expected Port default to be '8080', got '%s'", config.Port)
	}
	if config.Debug != true {
		t.Errorf("Expected Debug from env to be true, got %v", config.Debug)
	}
	if config.Audience != "partial-audience" {
		t.Errorf("Expected Audience from env to be 'partial-audience', got '%s'", config.Audience)
	}
	if config.Issuer != "heimdall" {
		t.Errorf("Expected Issuer default to be 'heimdall', got '%s'", config.Issuer)
	}
}

func TestLoadConfig_EmptyEnvironmentVariables(t *testing.T) {
	// Save original flags
	originalFlags := saveFlags()
	defer restoreFlags(originalFlags)

	// Clear environment variables first
	clearEnvVars()
	defer clearEnvVars()

	// Create new flag set to avoid conflicts
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	// Set empty environment variables (should use defaults)
	os.Setenv("HOST", "")
	os.Setenv("PORT", "")
	os.Setenv("DEBUG", "")
	os.Setenv("JWKS_URL", "")

	config := LoadConfig()

	// Should use defaults since env vars are empty
	if config.Host != "0.0.0.0" {
		t.Errorf("Expected Host default for empty env, got '%s'", config.Host)
	}
	if config.Port != "8080" {
		t.Errorf("Expected Port default for empty env, got '%s'", config.Port)
	}
	if config.Debug != false {
		t.Errorf("Expected Debug default for empty env, got %v", config.Debug)
	}
	if config.JWKSUrl != "http://heimdall:4457/.well-known/jwks" {
		t.Errorf("Expected JWKSUrl default for empty env, got '%s'", config.JWKSUrl)
	}
}

func TestLoadConfig_DebugEnvironmentVariableBehavior(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"debug_true", "true", true},
		{"debug_1", "1", true},
		{"debug_yes", "yes", true},
		{"debug_false", "false", false}, // "false" should set debug to false
		{"debug_0", "0", false},         // "0" should set debug to false
		{"debug_empty", "", false},      // Empty value uses default (false)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original flags
			originalFlags := saveFlags()
			defer restoreFlags(originalFlags)

			// Clear environment variables first
			clearEnvVars()
			defer clearEnvVars()

			// Create new flag set to avoid conflicts
			flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

			if tt.value != "" {
				os.Setenv("DEBUG", tt.value)
			}

			config := LoadConfig()
			if config.Debug != tt.expected {
				t.Errorf("Expected Debug to be %v for value '%s', got %v", tt.expected, tt.value, config.Debug)
			}
		})
	}
}

func TestConfig_StructFields(t *testing.T) {
	config := &Config{}

	// Test that all fields can be set
	config.Host = "test-host"
	config.Port = "test-port"
	config.Debug = true
	config.JWKSUrl = "test-jwks"
	config.Audience = "test-audience"
	config.Issuer = "test-issuer"
	config.NATSUrl = "test-nats"

	// Verify all fields were set correctly
	if config.Host != "test-host" {
		t.Errorf("Host field not set correctly")
	}
	if config.Port != "test-port" {
		t.Errorf("Port field not set correctly")
	}
	if config.Debug != true {
		t.Errorf("Debug field not set correctly")
	}
	if config.JWKSUrl != "test-jwks" {
		t.Errorf("JWKSUrl field not set correctly")
	}
	if config.Audience != "test-audience" {
		t.Errorf("Audience field not set correctly")
	}
	if config.Issuer != "test-issuer" {
		t.Errorf("Issuer field not set correctly")
	}
	if config.NATSUrl != "test-nats" {
		t.Errorf("NATSUrl field not set correctly")
	}
}

func TestLoadConfig_MultipleCallsConsistency(t *testing.T) {
	// Save original flags
	originalFlags := saveFlags()
	defer restoreFlags(originalFlags)

	// Clear environment variables first
	clearEnvVars()
	defer clearEnvVars()

	// Create new flag set to avoid conflicts
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	os.Setenv("HOST", "consistency-test")
	os.Setenv("PORT", "7777")

	config1 := LoadConfig()

	// Create another flag set
	flag.CommandLine = flag.NewFlagSet("test2", flag.ContinueOnError)
	config2 := LoadConfig()

	// Both calls should return the same values
	if config1.Host != config2.Host {
		t.Errorf("Inconsistent Host values: %s vs %s", config1.Host, config2.Host)
	}
	if config1.Port != config2.Port {
		t.Errorf("Inconsistent Port values: %s vs %s", config1.Port, config2.Port)
	}
	if config1.Debug != config2.Debug {
		t.Errorf("Inconsistent Debug values: %v vs %v", config1.Debug, config2.Debug)
	}
}

// Helper function to clear all environment variables used by config
func clearEnvVars() {
	os.Unsetenv("HOST")
	os.Unsetenv("PORT")
	os.Unsetenv("DEBUG")
	os.Unsetenv("JWKS_URL")
	os.Unsetenv("AUDIENCE")
	os.Unsetenv("ISSUER")
	os.Unsetenv("NATS_URL")
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Standard Go boolean values
		{"true", "true", true},
		{"false", "false", false},
		{"TRUE", "TRUE", true},
		{"FALSE", "FALSE", false},
		{"True", "True", true},
		{"False", "False", false},
		{"1", "1", true},
		{"0", "0", false},

		// Extended boolean values
		{"yes", "yes", true},
		{"no", "no", false},
		{"YES", "YES", true},
		{"NO", "NO", false},
		{"Yes", "Yes", true},
		{"No", "No", false},
		{"y", "y", true},
		{"n", "n", false},
		{"Y", "Y", true},
		{"N", "N", false},
		{"on", "on", true},
		{"off", "off", false},
		{"ON", "ON", true},
		{"OFF", "OFF", false},

		// Values with whitespace
		{"  true  ", "  true  ", true},
		{"  false  ", "  false  ", false},
		{"  yes  ", "  yes  ", true},
		{"  no  ", "  no  ", false},

		// Invalid values (should return false)
		{"empty", "", false},
		{"invalid", "invalid", false},
		{"random", "random", false},
		{"2", "2", false},
		{"-1", "-1", false},
		{"maybe", "maybe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBool(tt.input)
			if result != tt.expected {
				t.Errorf("parseBool(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
