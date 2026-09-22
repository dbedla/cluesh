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

	// mainCfg (model tag, clipboard mode, colors) is consumed as the
	// provider/tools wiring lands; validation of the config already happened.
	_, sysPrompt, err := cluesh.LoadConfig()
	if err != nil {
		panic(err)
	}

	provider, err := cluesh.BuildOpenRouterProvider(rellm.Model("z-ai/glm-5.3-flash"))
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

