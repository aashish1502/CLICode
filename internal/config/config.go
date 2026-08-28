package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DefaultLanguage string `json:"defaultLanguage"`
	AuthToken       string `json:"authToken"`

	// Languages is the set the editor lets you write in, by canonical id
	// ("python", "kotlin"). It is a property of the judge, not of any one
	// problem, so it is deliberately independent of which languages ship
	// starter code. Empty means languages.DefaultIDs(); the server's
	// capabilities response will supply it once the API exists.
	Languages []string `json:"languages"`
}

func Default() Config {
	return Config{
		DefaultLanguage: "python",
		AuthToken:       "",
		Languages:       nil,
	}
}

func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".clicode"), nil
}

func Load() (Config, error) {
	d, err := dir()
	if err != nil {
		return Default(), err
	}
	data, err := os.ReadFile(filepath.Join(d, "config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Default(), err
	}
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), err
	}
	return cfg, nil
}

func Save(c Config) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, "config.json"), data, 0600)
}
