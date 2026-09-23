package cluesh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLLMListMarked(t *testing.T) {
	ps, _, err := LoadProviders(t.TempDir()) // creates default provider templates
	assert.NoError(t, err)

	// default templates: or-glm53flash is openrouter's single model
	assert.Equal(t, []string{
		"openrouter.json",
		"\tor-glm53flash\tz-ai/glm-5.3-flash\ttemp=0.4 reasoning=low\t(default)",
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
		"\tor-glm53flash\tz-ai/glm-5.3-flash\ttemp=0.4 reasoning=low",
		"openai.json",
		"\toai-luna\tgpt-5.6-luna\treasoning=low",
		"lmstudio.json",
		"\tlms-gemma\tgoogle/gemma-4-26b-a4b",
	}, ps.LLMListMarked("no-such-tag"))
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
	assert.Equal(t, "temp=0.4 reasoning=low", ModelConfigData{Temperature: &temp, Reasoning: "low"}.Sampling())
	assert.Equal(t, "temp=0.4", ModelConfigData{Temperature: &temp}.Sampling())
	assert.Equal(t, "reasoning=high", ModelConfigData{Reasoning: "high"}.Sampling())
	assert.Equal(t, "", ModelConfigData{}.Sampling()) // unset → nothing sent
}
