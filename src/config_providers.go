package cluesh

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dbedla/rellm/pkg/rellm"
	"github.com/joho/godotenv"
)

// ProvidersDir is the subdirectory of ConfigDir holding one JSON file per
// provider type: openrouter.json, openai.json, lmstudio.json.
const ProvidersDir = "providers"

// ModelConfigData is one configurable model: Tag is the short user-facing label used
// with -llm and in default_model_tag, Name is the provider-internal model id.
// Sampling parameters are per model, because support is a property of the
// model, not the provider: an unset field means the parameter is not sent
// (e.g. OpenAI reasoning models reject temperature).
type ModelConfigData struct {
	Tag         string   `json:"tag"`
	Name        string   `json:"name"`
	Temperature *float64 `json:"temperature,omitempty"` // unset → not sent
	Reasoning   string   `json:"reasoning,omitempty"`   // none|low|medium|high, empty → not sent
}

// Sampling renders the model's non-default parameters for --llm-list.
func (m ModelConfigData) Sampling() string {
	s := ""
	if m.Temperature != nil {
		s += fmt.Sprintf("temp=%g ", *m.Temperature)
	}
	if m.Reasoning != "" {
		s += "reasoning=" + m.Reasoning
	}
	return strings.TrimRight(s, " ")
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
	AllModels() []ModelConfigData

	// SysPromptExtension returns text appended to the sys prompt when this
	// provider is in use, e.g. local models that ignore text.format get the
	// JSON schema embedded in the prompt. "" = nothing to add.
	SysPromptExtension() string
}

// findModel returns the model name for a tag within one provider's list.
func findModel(models []ModelConfigData, tag string) (string, bool) {
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
	APIKeyEnv string            `json:"api_key_env"` // read key from this environment variable
	APIKey    string            `json:"api_key"`     // inline key (least safe)
	Models    []ModelConfigData `json:"models"`
	// PromptExtension is appended to the sys prompt when this provider is
	// used.
	PromptExtension string `json:"sys_prompt_extension,omitempty"`
}

var _ ProviderConfig = (*OpenRouterConfig)(nil)

func (c *OpenRouterConfig) FileName() string             { return "openrouter.json" }
func (c *OpenRouterConfig) AllModels() []ModelConfigData { return c.Models }

// SysPromptExtension: whatever sys_prompt_extension holds in the config
// (empty by default)
func (c *OpenRouterConfig) SysPromptExtension() string { return c.PromptExtension }

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
	APIKeyEnv string            `json:"api_key_env"`
	APIKey    string            `json:"api_key"`
	Models    []ModelConfigData `json:"models"`
	// PromptExtension is appended to the sys prompt when this provider is
	// used. Empty by default
	PromptExtension string `json:"sys_prompt_extension,omitempty"`
}

var _ ProviderConfig = (*OpenAIConfig)(nil)

func (c *OpenAIConfig) FileName() string             { return "openai.json" }
func (c *OpenAIConfig) AllModels() []ModelConfigData { return c.Models }

// SysPromptExtension: whatever sys_prompt_extension holds in the config
// (empty by default).
func (c *OpenAIConfig) SysPromptExtension() string { return c.PromptExtension }

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
	BaseURL string            `json:"base_url"` // e.g. http://127.0.0.1
	Port    string            `json:"port"`     // e.g. 1234
	Models  []ModelConfigData `json:"models"`
	// PromptExtension is appended to the sys prompt when this provider is
	// used. LM Studio does not reliably pass text.format to local models,
	// so the generated default contains the JSON schema; edit freely.
	PromptExtension string `json:"sys_prompt_extension"`
}

var _ ProviderConfig = (*LMStudioConfig)(nil)

func (c *LMStudioConfig) FileName() string             { return "lmstudio.json" }
func (c *LMStudioConfig) AllModels() []ModelConfigData { return c.Models }

// SysPromptExtension returns the text appended to the sys prompt when this
// provider is in use — straight from the sys_prompt_text config field, so
// the file is the source of truth the user can edit.
func (c *LMStudioConfig) SysPromptExtension() string {
	return c.PromptExtension
}

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
	temp := 0.4
	return OpenRouterConfig{
		APIKeyEnv: "OPENROUTER_API_KEY",
		Models: []ModelConfigData{{
			Tag: "or-glm53flash", Name: "z-ai/glm-5.3-flash",
			Temperature: &temp, Reasoning: "low",
		}},
	}
}

func DefaultOpenAIConfig() OpenAIConfig {
	return OpenAIConfig{
		APIKeyEnv: "OPENAI_API_KEY",
		Models: []ModelConfigData{{
			Tag: "oai-luna", Name: "gpt-5.6-luna", Reasoning: "low",
			// no Temperature: OpenAI reasoning models reject the parameter
		}},
	}
}

func DefaultLMStudioConfig() LMStudioConfig {
	return LMStudioConfig{
		BaseURL:         "http://127.0.0.1",
		Port:            "1234",
		Models:          []ModelConfigData{{Tag: "lms-gemma", Name: "google/gemma-4-26b-a4b"}},
		PromptExtension: lmsSysPromptExtension(),
	}
}

// lmsSysPromptExtension builds the schema-in-prompt text (pattern from
// rellm's e2e_format_text_agent) written into the generated lmstudio.json.
func lmsSysPromptExtension() string {
	schema, err := json.Marshal(CommandTextFormat().Schema)
	if err != nil {
		return ""
	}
	return "\nOutput schema:\n" + string(schema) +
		"\nONLY parsable json string allowed as result, no additional markdown formatting"
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
func LoadProviders(baseDir string) (Providers, []string, error) {
	dir := ProvidersPath(baseDir)
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
	if err := checkModelSampling(ps.list); err != nil {
		return Providers{}, nil, err
	}
	return ps, created, nil
}

// checkModelSampling validates per-model parameters: reasoning must be an
// exact rellm enum value (temperature is a free float, not validated).
func checkModelSampling(list []ProviderConfig) error {
	for _, pc := range list {
		for _, m := range pc.AllModels() {
			if _, err := ReasoningEffort(m.Reasoning); err != nil && m.Reasoning != "" {
				return fmt.Errorf("%s: model %q: %w", pc.FileName(), m.Tag, err)
			}
		}
	}
	return nil
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

// Resolved is what Build returns for one tag: everything needed to run
// the agent — provider, sys prompt extension, and the model's own
// sampling parameters.
type MetaProvider struct {
	Provider     rellm.Provider
	SysPromptExt string
	ModelConfig  ModelConfigData
}

// Build resolves a tag to a ready provider: ask each provider in order.
// nil,nil from an implementation means "not my tag, ask the next one".
func (ps Providers) Build(tag string) (MetaProvider, error) {
	for _, pc := range ps.list {
		provider, err := pc.BuildForTag(tag)
		if err != nil {
			return MetaProvider{}, err // tag was mine but building failed — stop here
		}
		if provider != nil {
			modelConfig, _ := findModelByTag(ps.list, tag)
			return MetaProvider{Provider: provider, SysPromptExt: pc.SysPromptExtension(), ModelConfig: modelConfig}, nil
		}
	}
	return MetaProvider{}, fmt.Errorf("unknown model tag %q — run cluesh --llm-list to see configured models", tag)
}

// findModelByTag returns the configured Model for a tag across all providers.
// Tag uniqueness is enforced by LoadProviders, so at most one exists.
func findModelByTag(list []ProviderConfig, tag string) (ModelConfigData, bool) {
	for _, pc := range list {
		for _, m := range pc.AllModels() {
			if m.Tag == tag {
				return m, true
			}
		}
	}
	return ModelConfigData{}, false
}

// LLMList renders every configured tag for --llm-list, grouped by provider
// file: one header line per provider, tab-separated model lines below.
func (ps Providers) LLMList() []string {
	lines := []string{}
	for _, pc := range ps.list {
		lines = append(lines, pc.FileName())
		for _, m := range pc.AllModels() {
			line := "\t" + m.Tag + "\t" + m.Name
			if s := m.Sampling(); s != "" {
				line += "\t" + s
			}
			lines = append(lines, line)
		}
	}
	return lines
}

// LLMListMarked wraps LLMList, appending a (default) marker to the line of
// the default tag (default_model_tag); an unconfigured default stays unmarked.
func (ps Providers) LLMListMarked(defaultTag string) []string {
	lines := ps.LLMList()
	for i, line := range lines {
		fields := strings.SplitN(strings.TrimLeft(line, "\t"), "\t", 2)
		if fields[0] == defaultTag {
			lines[i] = line + "\t(default)"
		}
	}
	return lines
}

// ── first-run file generation ─────────────────────────────────────────

// ProvidersPath returns baseDir/providers.
func ProvidersPath(baseDir string) string {
	return filepath.Join(baseDir, ProvidersDir)
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
