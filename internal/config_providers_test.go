package cluesh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dbedla/rellm/pkg/rellm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMListMarked(t *testing.T) {
	ps, _, err := LoadProviders(t.TempDir()) // creates default provider templates
	assert.NoError(t, err)

	// default templates: or-glm53flash is openrouter's single model
	assert.Equal(t, []string{
		"openrouter.json",
		"\tor-glm53flash\tz-ai/glm-5.3-flash\ttemperature=0.4 reasoning=low\t(default)",
		"openai.json",
		"\toai-luna\tgpt-5.6-luna\treasoning=low",
		"lmstudio.json",
		"\tlms-gemma\tgoogle/gemma-4-26b-a4b",
	}, ps.LLMListMarked("or-glm53flash"))
}

func TestLLMListMarkedUnknownDefault(t *testing.T) {
	ps, _, err := LoadProviders(t.TempDir())
	assert.NoError(t, err)

	// unknown default tag: rendered, nothing marked
	assert.Equal(t, []string{
		"openrouter.json",
		"\tor-glm53flash\tz-ai/glm-5.3-flash\ttemperature=0.4 reasoning=low",
		"openai.json",
		"\toai-luna\tgpt-5.6-luna\treasoning=low",
		"lmstudio.json",
		"\tlms-gemma\tgoogle/gemma-4-26b-a4b",
	}, ps.LLMListMarked("no-such-tag"))
}

func TestLoadProvidersRejectsBadReasoning(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"unknown value", "bogus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			_, _, err := LoadProviders(baseDir)
			require.NoError(t, err)

			// replace the generated "reasoning": "low" with a bad value
			path := filepath.Join(ProvidersPath(baseDir), "openrouter.json")
			data, err := os.ReadFile(path)
			assert.NoError(t, err)
			out := strings.Replace(string(data),
				`"reasoning": "low"`,
				fmt.Sprintf("%q: %q", "reasoning", tt.value), 1)
			assert.NoError(t, os.WriteFile(path, []byte(out), 0o600))

			_, _, err = LoadProviders(baseDir)
			assert.Error(t, err)
		})
	}
}

func TestReasoningEffortMapping(t *testing.T) {
	for s, want := range map[string]rellm.ReasoningEffort{
		"none":   rellm.ReasoningEffortNone,
		"low":    rellm.ReasoningEffortLow,
		"medium": rellm.ReasoningEffortMedium,
		"high":   rellm.ReasoningEffortHigh,
	} {
		got, err := ReasoningEffort(s)
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := ReasoningEffort("bogus")
	assert.Error(t, err)
	_, err = ReasoningEffort("")
	assert.Error(t, err)
}

func TestResolveAPIKey(t *testing.T) {
	// env var wins over inline
	t.Setenv("OPENROUTER_API_KEY", "from-env")
	k, err := resolveAPIKey("openrouter", "OPENROUTER_API_KEY", "inline")
	assert.NoError(t, err)
	assert.Equal(t, "from-env", k)

	// env var unset → inline fallback
	t.Setenv("OPENROUTER_API_KEY", "")
	k, err = resolveAPIKey("openrouter", "OPENROUTER_API_KEY", "inline")
	assert.NoError(t, err)
	assert.Equal(t, "inline", k)

	// both unset → error naming the env var
	_, err = resolveAPIKey("openrouter", "OPENROUTER_API_KEY", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OPENROUTER_API_KEY")

	// no env name configured → inline directly
	k, err = resolveAPIKey("openrouter", "", "inline-only")
	assert.NoError(t, err)
	assert.Equal(t, "inline-only", k)

	// nothing configured at all
	_, err = resolveAPIKey("openrouter", "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no api key configured")
}

func TestBuildByTag(t *testing.T) {
	ps, _, err := LoadProviders(t.TempDir())
	assert.NoError(t, err)

	// LM Studio needs no api key → resolvable fully offline
	resolved, err := ps.Build("lms-gemma")
	assert.NoError(t, err)
	assert.NotNil(t, resolved.Provider)
	assert.Equal(t, "lms-gemma", resolved.ModelConfig.Tag)

	_, err = ps.Build("no-such-tag")
	assert.Error(t, err)
}

func TestModelSampling(t *testing.T) {
	temp := 0.4
	assert.Equal(t, "temperature=0.4 reasoning=low", ModelConfigData{Temperature: &temp, Reasoning: "low"}.Sampling())
	assert.Equal(t, "temperature=0.4", ModelConfigData{Temperature: &temp}.Sampling())
	assert.Equal(t, "reasoning=high", ModelConfigData{Reasoning: "high"}.Sampling())
	assert.Equal(t, "", ModelConfigData{}.Sampling()) // unset → nothing sent
}
