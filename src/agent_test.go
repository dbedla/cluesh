package cluesh

import (
	"encoding/json"
	"testing"

	"github.com/dbedla/rellm/pkg/rellm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageSummary(t *testing.T) {
	step := func(js string) rellm.StepStat {
		var u rellm.ResponsesAPIUsage
		if err := json.Unmarshal([]byte(js), &u); err != nil {
			t.Fatal(err)
		}
		return rellm.StepStat{APIUsage: u}
	}
	report := rellm.Report{StepsStats: []rellm.StepStat{
		step(`{"input_tokens":300,"output_tokens":30,"total_tokens":330,
			"input_tokens_details":{"cached_tokens":256},
			"output_tokens_details":{"reasoning_tokens":10},"cost":0.00004578}`),
		step(`{"input_tokens":70,"output_tokens":12,"total_tokens":82}`),
	}}
	assert.Equal(t,
		"info: tokens: in 370 (cached 256), out 42 (reasoning 10), total 412, cost $0.00004578",
		UsageSummary(report))

	// no stats → empty line, nothing printed
	assert.Equal(t, "", UsageSummary(rellm.Report{}))
}

func TestNewAgentDefaults(t *testing.T) {
	// LM Studio builds offline; zero MaxSteps and empty SysPrompt fall back
	// to DefaultMaxAgentSteps / ProgramSysPrompt instead of erroring.
	p, err := rellm.NewLMStudioProvider(rellm.Model("m"), "http://127.0.0.1", "1234")
	require.NoError(t, err)

	agent, err := NewAgent(AgentConfig{Provider: p, Conversation: rellm.NewInMemoryConversation()})
	require.NoError(t, err)
	require.NotNil(t, agent)
}
