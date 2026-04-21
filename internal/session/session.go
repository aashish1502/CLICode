package session

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Session struct {
	LastProblemID int `json:"lastProblemID"`
}

func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".clicode"), nil
}

func Load() (Session, error) {
	d, err := dir()
	if err != nil {
		return Session{}, err
	}
	data, err := os.ReadFile(filepath.Join(d, "session.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Session{}, nil
		}
		return Session{}, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}, err
	}
	return s, nil
}

func Save(s Session) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, "session.json"), data, 0600)
}
