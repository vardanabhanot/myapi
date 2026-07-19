package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Environments persist in the config dir root. History lives in its own
// history/ subdir there — ListHistory lists every .json in its dir and
// ClearHistory deletes them all, so they must not share a directory.

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

// ResolveEnv returns a deep copy with {{var}} placeholders substituted in
// the same fields SendRequest substitutes at send time. Used for codegen so
// the emitted snippet is runnable as-is.
func (r *Request) ResolveEnv() *Request {
	c := r.Clone()
	c.URL = ApplyEnv(c.URL)

	applyRows := func(rows *[]FormType) {
		if rows == nil {
			return
		}
		for i, row := range *rows {
			(*rows)[i].Key = ApplyEnv(row.Key)
			(*rows)[i].Value = ApplyEnv(row.Value)
		}
	}
	applyRows(c.Headers)
	applyRows(c.QueryParams)
	applyRows(c.Body.Form)

	c.Body.Json = ApplyEnv(c.Body.Json)
	c.Body.Xml = ApplyEnv(c.Body.Xml)
	c.Body.Text = ApplyEnv(c.Body.Text)

	if c.Auth != nil {
		c.Auth.BasicUser = ApplyEnv(c.Auth.BasicUser)
		c.Auth.BasicPass = ApplyEnv(c.Auth.BasicPass)
		c.Auth.BearerAuth = ApplyEnv(c.Auth.BearerAuth)
		c.Auth.APIKeyName = ApplyEnv(c.Auth.APIKeyName)
		c.Auth.APIKeyValue = ApplyEnv(c.Auth.APIKeyValue)
	}

	return c
}

// configFile resolves (and ensures) the app's config dir for a settings
// file — environments, collections. History lives in the history/ subdir.
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
