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

	if parseParams.Help {
		cluesh.PrintHelp(os.Stdout, baseDir)
		return
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

	modelTag := mainCfg.DefaultModelTag
	if parseParams.LLMTag != "" {
		modelTag = parseParams.LLMTag
	}

	metaProvider, err := providers.Build(modelTag)
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
		Provider:     metaProvider.Provider,
		Conversation: rellm.NewFilesystemConversation(convPath),
		SysPrompt:    sysPrompt + metaProvider.SysPromptExt,
		MaxSteps:     10,
	}

	agent, err := cluesh.NewAgent(cfg)
	if err != nil {
		exit(err)
	}

	q := parseParams.Prompt

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(mainCfg.ExecutionTimeoutMinutes)*time.Minute)
	defer cancel()

	prompt, err := buildPrompt(q, metaProvider.ModelConfig)
	if err != nil {
		exit(err)
	}

	report, err := waitWithProgressbar(ctx, agent, prompt)
	if err != nil {
		exit(err)
	}

	cmd, err := cluesh.ParseAgentResult(report)
	if err != nil {
		exit(err)
	}

	cluesh.Print(cluesh.ConsolByName(mainCfg.Colors), cmd)
	printUsage(report)

	if cluesh.ShouldCopy(mainCfg.PutCmdInClipboard, cmd.RedOnly) {
		if err := cluesh.CopyToClipboard(cmd.FinalCommand); err != nil {
			fmt.Fprintln(os.Stderr, "clipboard:", err)
		} else {
			fmt.Println("\ninfo: command in clipboard")
		}
	}
}

func buildPrompt(q string, m cluesh.ModelConfigData) (*rellm.Prompt, error) {
	// per-model sampling: unset fields are simply not sent (some models, e.g.
	// OpenAI reasoning models, reject unsupported parameters with a 400)
	promptBuilder := rellm.NewPromptBuilder().
		WithMessage(q)
	if m.Temperature != nil {
		promptBuilder = promptBuilder.WithTemperature(*m.Temperature)
	}
	if m.Reasoning != "" {
		effort, err := cluesh.ReasoningEffort(m.Reasoning)
		if err != nil {
			return nil, err
		}
		promptBuilder = promptBuilder.WithReasoning(effort)
	}
	return promptBuilder.Build()
}

// exit prints the error to stderr and terminates with status 1.
func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// printUsage prints the token/cost summary line; silent without stats.
func printUsage(report rellm.Report) {
	if line := cluesh.UsageSummary(report); line != "" {
		fmt.Println(line)
	}
}

func waitWithProgressbar(ctx context.Context, agent *rellm.Agent, p *rellm.Prompt) (rellm.Report, error) {

	reportChannel := make(chan agentReport)

	fn := func() {
		report, err := agent.Execute(ctx, p)
		ar := agentReport{
			report: report,
			err:    err,
		}
		reportChannel <- ar
	}
	go fn()

	ticker := time.NewTicker(1600 * time.Millisecond)
	defer ticker.Stop()
	defer fmt.Println()

	for {
		select {
		case ar := <-reportChannel:
			return ar.report, ar.err
		case <-ticker.C:
			fmt.Print("=")
		}
	}

}

type agentReport struct {
	report rellm.Report
	err    error
}
