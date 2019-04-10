// Package config reads configuration from the Scramr config file
package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Hostname       string              `yaml:"hostname"`
	ListenAddr     string              `yaml:"listen_address"`
	ListenPort     int                 `yaml:"listen_port"`
	ForwardMap     map[string][]string `yaml:"forwards"`
	ServerTimeouts map[string]string   `yaml:"server_timeouts"`
	ClientTimeouts map[string]string   `yaml:"client_timeouts"`
	Timeouts       timeouts
	Forward        map[string]string
}

type timeouts struct {
	ServerRx, ServerWr, ClientConn, ClientRx, ClientWr time.Duration
}

// func Read opens the specified config file and reads it into the Config struct
func Read(f string) (*Config, error) {
	// Default config options
	c := &Config{
		ListenPort: 25,
		ListenAddr: "0.0.0.0",
		ServerTimeouts: map[string]string{
			"read":  "30s",
			"write": "30s",
		},
		ClientTimeouts: map[string]string{
			"connect": "30s",
			"read":    "30s",
			"write":   "30s",
		},
	}

	// Set default hostname from the OS
	var err error
	c.Hostname, err = os.Hostname()
	if err != nil {
		log.Printf("WARNING: Could not determine hostname: %v", err)
	}

	d, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("Could not read config file: %v", err)
	}

	err = yaml.Unmarshal([]byte(d), &c)
	if err != nil {
		return nil, fmt.Errorf("Error in config file: %v", err)
	}

	err = c.compileForwards()
	if err != nil {
		return nil, err
	}

	err = c.collectTimeouts()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// func compileForwards creates a map of emailaddress to target
// This is so we can quickly check the map when receiving new mail
func (c *Config) compileForwards() error {
	if c.ForwardMap == nil {
		return fmt.Errorf("No forwards configured")
	}
	c.Forward = map[string]string{}

	for target, addrs := range c.ForwardMap {
		for _, addr := range addrs {
			c.Forward[addr] = target
		}
	}

	return nil
}

// func collectTimeouts parses string into time.Duration types
func (c *Config) collectTimeouts() error {
	// Parse all timeouts into durations
	c.Timeouts.ServerRx = parseTime("server_timeout[read]", c.ServerTimeouts["read"])
	c.Timeouts.ServerWr = parseTime("server_timeout[write]", c.ServerTimeouts["write"])
	c.Timeouts.ClientRx = parseTime("client_timeout[read]", c.ClientTimeouts["read"])
	c.Timeouts.ClientWr = parseTime("client_timeout[write]", c.ClientTimeouts["write"])
	c.Timeouts.ClientConn = parseTime("client_timeout[connect]", c.ClientTimeouts["connect"])

	// TODO: Ensure timeouts configured make sense (client_timeout < server_timeout)

	return nil
}

// func parseTime takes a string and converts it to a time.Duration
func parseTime(i, s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("Could not parse timeout %v: %v", i, err)
	}
	if d <= 0 {
		log.Fatalf("%v must be above 0 seconds (currently set to: %v)", i, d)
	}
	return d
}
