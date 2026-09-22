package main

import (
	cluesh "cluesh/src"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dbedla/rellm/pkg/rellm"
)

func main() {

	provider, err := cluesh.BuildOpenRouterProvider(rellm.Model("z-ai/glm-5.3-flash"))
	if err != nil {
		panic(err)
	}

	cfg := cluesh.AgentConfig{
		Provider:     provider,
		Conversation: rellm.NewInMemoryConversation(),
		SysPrompt:    defaultSysPrompt,
		MaxSteps:     10,
	}

	agent, err := cluesh.NewAgent(cfg)
	if err != nil {
		panic(err)
	}

	if len(os.Args) != 2 {
		panic("bad args number")
	}
	q := os.Args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	report, err := agent.Ask(ctx, q)
	if err != nil {
		panic(err)
	}

	fmt.Printf("raw: %s\n", report.Message)

	cmd, err := cluesh.ParseAgentResult(report)
	if err != nil {
		panic(err)
	}

	cluesh.Print(&cluesh.DarkMode{}, cmd)
}

const defaultSysPrompt = `You are a bash expert.
Your main task is to provide one final bash command (or bash command combination) which fulfills the requested demand.
The answer must be a oneline copy-paste ready bash command.
Prefer these commands where possible: find, grep, cut, sort, uniq, xargs, wc, head, cat, less, tail, wc, ls, tree.
No loops and ifs unless absolutely necessary.
Placeholder in example should be as short as possible.
subCommands must list EVERY command of the pipeline in order, including the first, each with ALL its arguments explained — e.g. "ls -la dir" → one subCommand "ls" with arguments -l, -a, dir; never leave subCommands empty.
Return only JSON matching the given schema.`
