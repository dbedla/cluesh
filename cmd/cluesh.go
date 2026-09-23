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

	provider, promptExtension, err := providers.Build(mainCfg.DefaultModelTag)
	if err != nil {
		exit(err)
	}

	cfg := cluesh.AgentConfig{
		Provider:     provider,
		Conversation: rellm.NewFilesystemConversation(cluesh.ConversationPath(baseDir)),
		SysPrompt:    sysPrompt + promptExtension,
		MaxSteps:     10,
	}

	agent, err := cluesh.NewAgent(cfg)
	if err != nil {
		exit(err)
	}

	if len(os.Args) != 2 {
		exit(fmt.Errorf("bad args number"))
	}
	q := os.Args[1]

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
