package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Environments persist in the user config dir, NOT the cache dir: history
// owns that directory (ListHistory lists every file in it and ClearHistory
// deletes them all).

type Environment struct {
	Name      string      `json:"Name"`
	Variables *[]FormType `json:"Variables"`
}

type EnvStore struct {
	Active string         `json:"Active"` // active env name; "" means none
	Envs   []*Environment `json:"Envs"`
}

var (
	envMu      sync.RWMutex
	activeVars map[string]string
)

// SetActiveVars swaps the variable set ApplyEnv substitutes from.
// Pass nil to disable substitution.
func SetActiveVars(vars map[string]string) {
	envMu.Lock()
	activeVars = vars
	envMu.Unlock()
}

// VarMap returns the checked, non-empty-key variables. Nil-safe so callers
// can chain store.ActiveEnv().VarMap().
func (e *Environment) VarMap() map[string]string {
	vars := make(map[string]string)
	if e == nil || e.Variables == nil {
		return vars
	}

	for _, v := range *e.Variables {
		if v.Checked && v.Key != "" {
			vars[v.Key] = v.Value
		}
	}

	return vars
}

// ActiveEnv resolves the store's active environment, nil when none.
func (s *EnvStore) ActiveEnv() *Environment {
	if s.Active == "" {
		return nil
	}

	for _, e := range s.Envs {
		if e.Name == s.Active {
			return e
		}
	}

	return nil
}

// ApplyEnv replaces every {{key}} with the active environment's value.
// Unknown keys stay literal, like Postman.
// ponytail: exact {{key}} only — no {{ key }} whitespace trimming.
func ApplyEnv(s string) string {
	if !strings.Contains(s, "{{") {
		return s
	}

	envMu.RLock()
	defer envMu.RUnlock()

	for k, v := range activeVars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}

	return s
}

// configFile resolves (and ensures) the app's config dir for a settings
// file — environments, collections. Distinct from the cache dir history owns.
func configFile(name string) (string, error) {
	dir, err := os.UserConfigDir()

	if err != nil {
		return "", err
	}

	myapiPath := filepath.Join(dir, "myapi")

	if err := os.MkdirAll(myapiPath, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(myapiPath, name), nil
}

// LoadEnvStore reads saved environments; empty store on any error.
func LoadEnvStore() *EnvStore {
	store := &EnvStore{}

	file, err := configFile("environments.json")
	if err != nil {
		return store
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return store
	}

	json.Unmarshal(content, store)

	return store
}

func SaveEnvStore(store *EnvStore) error {
	file, err := configFile("environments.json")

	if err != nil {
		return err
	}

	data, err := json.Marshal(store)

	if err != nil {
		return err
	}

	return os.WriteFile(file, data, 0o644)
}
