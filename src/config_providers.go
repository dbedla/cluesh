package cluesh

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dbedla/rellm/pkg/rellm"
	"github.com/joho/godotenv"
)

// ProvidersDir is the subdirectory of ConfigDir holding one JSON file per
// provider type: openrouter.json, openai.json, lmstudio.json.
const ProvidersDir = "providers"

// Model is one configurable model: Tag is the short user-facing label used
// with -llm and in default_model_tag, Name is the provider-internal model id.
type Model struct {
	Tag  string `json:"tag"`
	Name string `json:"name"`
}

// ProviderConfig is the contract every provider implementation fulfills.
// Each implementation owns exactly one file in ~/.cluesh/providers/.
type ProviderConfig interface {
	// FileName is the config file this provider reads (e.g. "openrouter.json").
	FileName() string

	// LoadOrCreate reads the provider's own file into itself. A missing
	// file is generated from the provider's default template first; created
	// reports that. One pass does ensure + load.
	LoadOrCreate(dir string) (created bool, err error)

	// BuildForTag: if the tag is defined in this provider, construct and
	// return the rellm provider. If the tag is not defined here, return
	// (nil, nil). Error only when the tag IS mine but building fails
	// (bad auth, invalid model, ...).
	BuildForTag(tag string) (rellm.Provider, error)

	// AllModels returns every configured model — used for tag listing and
	// conflict detection across providers.
	AllModels() []Model
}

// findModel returns the model name for a tag within one provider's list.
func findModel(models []Model, tag string) (string, bool) {
	for _, m := range models {
		if m.Tag == tag {
			return m.Name, true
		}
	}
	return "", false
}

// ── OpenRouter ────────────────────────────────────────────────────────

// OpenRouterConfig is the content of providers/openrouter.json.
type OpenRouterConfig struct {
	APIKeyEnv string  `json:"api_key_env"` // read key from this environment variable
	APIKey    string  `json:"api_key"`     // inline key (least safe)
	Models    []Model `json:"models"`
}

var _ ProviderConfig = (*OpenRouterConfig)(nil)

func (c *OpenRouterConfig) FileName() string   { return "openrouter.json" }
func (c *OpenRouterConfig) AllModels() []Model { return c.Models }

// LoadOrCreate reads providers/openrouter.json into itself; a missing file
// is generated from the default template first.
func (c *OpenRouterConfig) LoadOrCreate(dir string) (bool, error) {
	return loadOrCreate(filepath.Join(dir, c.FileName()), DefaultOpenRouterConfig(), c)
}

// BuildForTag builds the provider when tag belongs to OpenRouter.
func (c *OpenRouterConfig) BuildForTag(tag string) (rellm.Provider, error) {
	name, ok := findModel(c.Models, tag)
	if !ok {
		return nil, nil // not my tag
	}
	key, err := resolveAPIKey("openrouter", c.APIKeyEnv, c.APIKey)
	if err != nil {
		return nil, err
	}
	return rellm.NewOpenRouterProvider(key, rellm.Model(name))
}

// ── OpenAI ────────────────────────────────────────────────────────────

// OpenAIConfig is the content of providers/openai.json.
type OpenAIConfig struct {
	APIKeyEnv string  `json:"api_key_env"`
	APIKey    string  `json:"api_key"`
	Models    []Model `json:"models"`
}

var _ ProviderConfig = (*OpenAIConfig)(nil)

func (c *OpenAIConfig) FileName() string   { return "openai.json" }
func (c *OpenAIConfig) AllModels() []Model { return c.Models }

// LoadOrCreate reads providers/openai.json into itself; a missing file
// is generated from the default template first.
func (c *OpenAIConfig) LoadOrCreate(dir string) (bool, error) {
	return loadOrCreate(filepath.Join(dir, c.FileName()), DefaultOpenAIConfig(), c)
}

// BuildForTag builds the provider when tag belongs to OpenAI.
func (c *OpenAIConfig) BuildForTag(tag string) (rellm.Provider, error) {
	name, ok := findModel(c.Models, tag)
	if !ok {
		return nil, nil
	}
	key, err := resolveAPIKey("openai", c.APIKeyEnv, c.APIKey)
	if err != nil {
		return nil, err
	}
	return rellm.NewOpenAIProvider(key, rellm.Model(name))
}

// ── LM Studio ─────────────────────────────────────────────────────────

// LMStudioConfig is the content of providers/lmstudio.json.
type LMStudioConfig struct {
	BaseURL string  `json:"base_url"` // e.g. http://127.0.0.1
	Port    string  `json:"port"`     // e.g. 1234
	Models  []Model `json:"models"`
}

var _ ProviderConfig = (*LMStudioConfig)(nil)

func (c *LMStudioConfig) FileName() string   { return "lmstudio.json" }
func (c *LMStudioConfig) AllModels() []Model { return c.Models }

// LoadOrCreate reads providers/lmstudio.json into itself; a missing file
// is generated from the default template first.
func (c *LMStudioConfig) LoadOrCreate(dir string) (bool, error) {
	return loadOrCreate(filepath.Join(dir, c.FileName()), DefaultLMStudioConfig(), c)
}

// BuildForTag builds the provider when tag belongs to LM Studio.
// LM Studio needs no api key.
func (c *LMStudioConfig) BuildForTag(tag string) (rellm.Provider, error) {
	name, ok := findModel(c.Models, tag)
	if !ok {
		return nil, nil // not my tag
	}
	return rellm.NewLMStudioProvider(rellm.Model(name), c.BaseURL, c.Port)
}

// ── templates for first run ───────────────────────────────────────────

func DefaultOpenRouterConfig() OpenRouterConfig {
	return OpenRouterConfig{
		APIKeyEnv: "OPENROUTER_API_KEY",
		Models:    []Model{{Tag: "or-glm53flash", Name: "z-ai/glm-5.3-flash"}},
	}
}

func DefaultOpenAIConfig() OpenAIConfig {
	return OpenAIConfig{
		APIKeyEnv: "OPENAI_API_KEY",
		Models:    []Model{{Tag: "oai-luna", Name: "gpt-5.6-luna"}},
	}
}

func DefaultLMStudioConfig() LMStudioConfig {
	return LMStudioConfig{
		BaseURL: "http://127.0.0.1",
		Port:    "1234",
		Models:  []Model{{Tag: "lms-gemma", Name: "google/gemma-4-26b-a4b"}},
	}
}

// loadOrCreate is the shared file plumbing behind LoadOrCreate: the provider
// supplies path, template and unmarshal target — knowledge of file name and
// template stays with the implementation.
func loadOrCreate(path string, template, v any) (created bool, err error) {
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		if wErr := writeJSON(path, template); wErr != nil {
			return false, wErr
		}
		created = true
	} else if statErr != nil {
		return false, fmt.Errorf("cannot access %s: %w", path, statErr)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("cannot read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return false, fmt.Errorf("invalid json in %s: %w", path, err)
	}
	return created, nil
}

// ── registry ──────────────────────────────────────────────────────────

// Providers keeps the list of all provider implementations and routes
// tag lookups across them.
type Providers struct {
	list []ProviderConfig
}

// LoadProviders creates the provider list; every provider then loads its
// own file (creating it from its default template when missing) and tag
// conflicts are checked. One pass does ensure + load.
func LoadProviders() (Providers, []string, error) {
	dir, err := ProvidersPath()
	if err != nil {
		return Providers{}, nil, err
	}
	if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
		return Providers{}, nil, fmt.Errorf("cannot create %s: %w", dir, mkErr)
	}

	ps := Providers{list: []ProviderConfig{
		&OpenRouterConfig{},
		&OpenAIConfig{},
		&LMStudioConfig{},
	}}

	created := []string{}
	for _, pc := range ps.list {
		wasCreated, err := pc.LoadOrCreate(dir)
		if err != nil {
			return Providers{}, nil, err
		}
		if wasCreated {
			created = append(created, filepath.Join(dir, pc.FileName()))
		}
	}

	if err := checkTagConflicts(ps.list); err != nil {
		return Providers{}, nil, err
	}
	return ps, created, nil
}

// checkTagConflicts: every tag must be unique across all providers.
func checkTagConflicts(list []ProviderConfig) error {
	seen := map[string]string{} // tag → provider
	for _, pc := range list {
		for _, m := range pc.AllModels() {
			if prev, dup := seen[m.Tag]; dup {
				return fmt.Errorf("duplicate model tag %q (already used by %s)", m.Tag, prev)
			}
			seen[m.Tag] = pc.FileName()
		}
	}
	return nil
}

// Build resolves a tag to a ready provider: ask each provider in order.
// nil,nil from an implementation means "not my tag, ask the next one".
func (ps Providers) Build(tag string) (rellm.Provider, error) {
	for _, pc := range ps.list {
		provider, err := pc.BuildForTag(tag)
		if err != nil {
			return nil, err // tag was mine but building failed — stop here
		}
		if provider != nil {
			return provider, nil // tag was mine, built — done
		}
	}
	return nil, fmt.Errorf("unknown model tag %q — run cluesh --llm-list to see configured models", tag)
}

// LLMList renders every configured tag for --llm-list.
func (ps Providers) LLMList() []string {
	lines := []string{}
	for _, pc := range ps.list {
		for _, m := range pc.AllModels() {
			lines = append(lines, fmt.Sprintf("%-20s %-16s %s", m.Tag, pc.FileName(), m.Name))
		}
	}
	return lines
}

// ── first-run file generation ─────────────────────────────────────────

// ProvidersPath returns ~/.cluesh/providers.
func ProvidersPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ProvidersDir), nil
}

// ── auth helpers ──────────────────────────────────────────────────────

// resolveAPIKey reads the key: env var wins, then inline.
// Both unset is an error for key-requiring providers.
func resolveAPIKey(provider, envName, inline string) (string, error) {
	if envName != "" {
		if v, envErr := envAPIKey(provider, envName); envErr == nil {
			return v, nil
		} else if inline == "" {
			return "", envErr
		}
	}
	if inline != "" {
		return inline, nil
	}
	return "", fmt.Errorf("%s: no api key configured — set api_key_env or api_key in providers/%s.json", provider, provider)
}

// envAPIKey reads an api key from an environment variable (tolerating a
// missing .env file — the value may come from the shell itself).
func envAPIKey(provider, envName string) (string, error) {
	_ = godotenv.Load()
	if v := os.Getenv(envName); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("%s: api key not set — export %s or set api_key in providers/%s.json", provider, envName, provider)
}
