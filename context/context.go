package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Context struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ServerURL   string `json:"server_url"`
}

// GetServerURL returns the server URL based on the current context
func (c *Context) GetServerURL() (string, error) {
	if c.ServerURL == "" {
		return "http://localhost:4000", nil
	}

	ctx, err := LoadContext(c.Name)
	if err != nil {
		return "", err
	}

	return ctx.ServerURL, nil
}
func SaveContext(ctx Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home directory: %w", err)
	}

	dir := filepath.Join(home, ".sequin", "contexts")
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("could not create contexts directory: %w", err)
	}

	file := filepath.Join(dir, ctx.Name+".json")
	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal context: %w", err)
	}

	err = os.WriteFile(file, data, 0644)
	if err != nil {
		return fmt.Errorf("could not write context file: %w", err)
	}

	return nil
}

func LoadContext(name string) (*Context, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}

	file := filepath.Join(home, ".sequin", "contexts", name+".json")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read context file: %w", err)
	}

	var ctx Context
	err = json.Unmarshal(data, &ctx)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal context: %w", err)
	}

	return &ctx, nil
}
