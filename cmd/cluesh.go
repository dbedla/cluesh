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

	//keep this will be usefull in next round of test
	// if len(os.Args) == 1 {
	// 	printTest()
	// 	return
	// }

	mainCfg, sysPrompt, err := cluesh.LoadConfig()
	if err != nil {
		panic(err)
	}

	providers, created, err := cluesh.LoadProviders()
	for _, p := range created {
		fmt.Printf("created %s\n", p)
	}
	if err != nil {
		panic(err)
	}

	provider, err := providers.Build(mainCfg.DefaultModelTag)
	if err != nil {
		panic(err)
	}

	cfg := cluesh.AgentConfig{
		Provider:     provider,
		Conversation: rellm.NewInMemoryConversation(),
		SysPrompt:    sysPrompt,
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
