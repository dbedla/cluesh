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
	baseDir, err := cluesh.DefaultBaseDir()
	if err != nil {
		exit(err)
	}

	parseParams, err := cluesh.ParseParams(os.Args[1:])
	if err != nil {
		exit(err)
	}

	mainCfg, sysPrompt, err := cluesh.LoadConfig(baseDir)
	if err != nil {
		exit(err)
	}

	providers, created, err := cluesh.LoadProviders(baseDir)
	for _, p := range created {
		fmt.Printf("created %s\n", p)
	}
	if err != nil {
		exit(err)
	}

	if parseParams.LLMList {
		for _, line := range providers.LLMListMarked(mainCfg.DefaultModelTag) {
			fmt.Println(line)
		}
		return
	}

	provider, promptExtension, err := providers.Build(mainCfg.DefaultModelTag)
	if err != nil {
		exit(err)
	}

	convPath := cluesh.ConversationPath(baseDir)
	if !parseParams.Continue {
		if err := cluesh.StartFresh(convPath); err != nil {
			exit(err)
		}
	}

	cfg := cluesh.AgentConfig{
		Provider:     provider,
		Conversation: rellm.NewFilesystemConversation(convPath),
		SysPrompt:    sysPrompt + promptExtension,
		MaxSteps:     10,
	}

	agent, err := cluesh.NewAgent(cfg)
	if err != nil {
		exit(err)
	}

	q := parseParams.Prompt

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(mainCfg.ExecutionTimeoutMinutes)*time.Minute)
	defer cancel()

	report, err := agent.Ask(ctx, q)
	if err != nil {
		exit(err)
	}

	fmt.Printf("raw: %s\n", report.Message)

	cmd, err := cluesh.ParseAgentResult(report)
	if err != nil {
		exit(err)
	}

	cluesh.Print(cluesh.ConsolByName(mainCfg.Colors), cmd)
}

// exit prints the error to stderr and terminates with status 1.
func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
