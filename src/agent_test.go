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

// rawjson is a realistic agent-shaped answer — recovered from the old tp.go
// demo — used to test the critical parse path.
const rawjson = `{
  "final_command": "find . -type f -name '*.go' | xargs -n1 wc -l | awk '$1 > 10'",
  "sub_commands": [
    {
      "command": "find",
      "arguments": [
        {"argument": ".", "description": "start the search in the current directory"},
        {"argument": "-type", "description": "match only regular files"},
        {"argument": "f", "description": "value for -type: regular file"},
        {"argument": "-name", "description": "match files by pattern"},
        {"argument": "'*.go'", "description": "value for -name: files ending with the .go extension"}
      ]
    },
    {
      "command": "xargs",
      "arguments": [
        {"argument": "-n1", "description": "pass exactly one filename per invocation so wc runs once per file and never prints an aggregate total line"}
      ]
    },
    {
      "command": "wc",
      "arguments": [
        {"argument": "-l", "description": "count the lines of the received file, printing the count followed by the filename"}
      ]
    },
    {
      "command": "awk",
      "arguments": [
        {"argument": "'$1 > 10'", "description": "print only lines whose first field (the line count) is greater than 10, i.e. keep only files with more than 10 lines"}
      ]
    }
  ],
  "notes": "xargs -n1 avoids the 'total' summary line.",
  "read_only": true
}`

func TestParseAgentResult(t *testing.T) {
	cmd, err := ParseAgentResult(rellm.Report{Message: rawjson})
	require.NoError(t, err)
	assert.Equal(t, "find", cmd.SubCommands[0].Command)
	assert.Len(t, cmd.SubCommands, 4)
	assert.True(t, cmd.ReadOnly)
	assert.Equal(t, "find . -type f -name '*.go' | xargs -n1 wc -l | awk '$1 > 10'", cmd.FinalCommand)

	// non-JSON agent output fails with a wrapped error
	_, err = ParseAgentResult(rellm.Report{Message: "not json"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not valid JSON")
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
