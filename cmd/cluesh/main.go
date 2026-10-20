package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	cluesh "github.com/dbedla/cluesh/internal"

	"github.com/dbedla/rellm/pkg/rellm"
	"github.com/fatih/color"
)

func main() {
	baseDir, err := cluesh.DefaultBaseDir()
	if err != nil {
		exit(err)
	}

	if cluesh.IsFirstRun(baseDir) {
		firstRun(baseDir)
		return
	}

	parseParams, err := cluesh.ParseParams(os.Args[1:])
	if err != nil {
		exit(err)
	}

	if parseParams.Help {
		fmt.Print(cluesh.GetHelp(baseDir))
		return
	}

	mainCfg, sysPrompts, err := cluesh.LoadConfig(baseDir)
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

	q := parseParams.Prompt
	if !parseParams.Explain {
		err := checkDemandLimit(q, mainCfg.InputLimitInBytes)
		if err != nil {
			exit(err)
		}
	}
	// -e/--explain: argument wins, else $EDITOR on a terminal, else stdin
	// (file/pipe). Runs before StartFresh so a cancelled or failed explain
	// does not delete the saved conversation. The mode also selects the sys
	// prompt: sysprompt-explain.md vs sysprompt-cmd.md.
	sysPrompt := sysPrompts.Cmd
	if parseParams.Explain {
		sysPrompt = sysPrompts.Explain
		q, err = promptForExplainFlow(q, mainCfg.InputLimitInBytes)
		if err != nil {
			exit(err)
		}
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(mainCfg.ExecutionTimeoutMinutes)*time.Minute)
	defer cancel()

	prompt, err := buildPrompt(q, metaProvider.ModelConfig)
	if err != nil {
		exit(err)
	}

	printModelConfig(metaProvider.ModelConfig)
	report, err := waitWithProgressbar(ctx, agent, prompt)
	if err != nil {
		exit(err)
	}

	cmd, err := cluesh.ParseAgentResult(report)
	if err != nil {
		exit(err)
	}

	cluesh.Print(cluesh.ConsoleByName(mainCfg.Colors), cmd)
	printUsage(report)

	printLLMClaimOnCmd(cmd.ReadOnly)

	if cluesh.ShouldCopy(mainCfg.PutCmdInClipboard, cmd.ReadOnly) {
		if err := cluesh.CopyToClipboard(cmd.FinalCommand); err != nil {
			fmt.Fprintln(os.Stderr, "clipboard:", err)
		} else {
			fmt.Println("info: command in clipboard")
		}
	}
}

// checkDemandLimit enforces input_limit_in_bytes on the generation demand:
// anything bigger is a misdirected file dump, and the provider would only
// reject it after the tokens are spent.
func checkDemandLimit(q string, limitInBytes int) error {
	if len(q) > limitInBytes {
		return fmt.Errorf("demand is %d bytes, over the %d byte input_limit_in_bytes — it looks like a file, not a demand", len(q), limitInBytes)
	}
	return nil
}

// promptForExplainFlow resolves the command to explain — argument, else
// $EDITOR on a terminal, else stdin (file/pipe) — and enforces limitInBytes
// on every path: a bash command is a few KB at most, so anything bigger is a
// misdirected file, and reading it unbounded would load it all into memory
// before the provider rejects it.
func promptForExplainFlow(q string, limitInBytes int) (string, error) {
	if q == "" {
		var err error
		if isTerminal(os.Stdin) {
			q, err = readFromEditor(limitInBytes)
		} else {
			// +1 so an over-limit input is detected without reading it all
			var b []byte
			b, err = io.ReadAll(io.LimitReader(os.Stdin, int64(limitInBytes+1)))
			q = string(b)
		}
		if err != nil {
			return "", err
		}
		if q == "" {
			return "", errors.New("no command to explain")
		}
	}
	// one check covers all three input paths (argument, stdin, editor)
	if len(q) > limitInBytes {
		return "", fmt.Errorf("command to explain is %d bytes, over the %d byte input_limit_in_bytes — it looks like a file, not a command", len(q), limitInBytes)
	}
	// drift alarm: show what actually arrived, before paying for the call.
	// strconv.Quote neutralizes control/escape chars from untrusted input.
	fmt.Printf("info: explaining: %s\n", strconv.Quote(q))

	return q, nil
}

func firstRun(baseDir string) {
	if _, _, err := cluesh.LoadConfig(baseDir); err != nil {
		exit(err)
	}
	_, created, err := cluesh.LoadProviders(baseDir)
	if err != nil {
		exit(err)
	}
	for _, p := range created {
		fmt.Printf("created %s\n", p)
	}
	fmt.Println("\nFirst run: default configuration created. Finish configuration:")
	fmt.Println("  - set your API key in a provider file, e.g. providers/openrouter.json:")
	fmt.Println(`      "api_key_env": "OPENROUTER_API_KEY"`)
	fmt.Println("  - review config.json and pick your default model (default_model_tag)")
	fmt.Println(`Then run: cluesh "<demand>"`)
	fmt.Println("Details: cluesh --help / https://github.com/dbedla/cluesh/blob/main/README.md")
}

func printModelConfig(m cluesh.ModelConfigData) {
	fmt.Printf("\nModel: %s\n", m.Name)
	fmt.Printf("%s\n", m.Sampling())
}

func printLLMClaimOnCmd(noFileModification bool) {
	if noFileModification {
		fmt.Println("LLM claim: <no file modification>")
	} else {
		color.Red("LLM claim: <some file will be modified>")
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

// isTerminal reports whether f is attached to a terminal rather than a
// file or pipe. term.IsTerminal uses an isatty ioctl, so char devices like
// /dev/null are correctly reported as non-terminals.
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// readFromEditor opens $VISUAL, $EDITOR (fallback: vi on Unix, notepad on
// Windows) on a temp file and returns its contents after the editor exits.
// The file is removed after.
func readFromEditor(limitInBytes int) (_ string, finalError error) {
	f, err := os.CreateTemp("", "cluesh-*.sh")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer func() {
		rmErr := os.Remove(path)
		finalError = errors.Join(finalError, rmErr)
	}()

	if err := f.Close(); err != nil {
		return "", err
	}

	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	// Windows has no sh; cmd /c gives the same "editor value may contain
	// arguments" behavior. WSL users run the Linux build, where this
	// branch never executes.
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		if editor == "" {
			editor = "notepad"
		}
		cmd = exec.Command("cmd", "/c", editor, path)
	} else {
		if editor == "" {
			editor = "vi"
		}
		// run via sh so $EDITOR values with arguments (e.g. "code -w") work
		cmd = exec.Command("sh", "-c", editor+" \"$@\"", editor, path)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q: %w", editor, err)
	}

	// refuse an oversized temp file before loading it into memory
	if info, statErr := os.Stat(path); statErr == nil && info.Size() > int64(limitInBytes) {
		return "", fmt.Errorf("editor content is %d bytes, over the %d byte input_limit_in_bytes — it looks like a file, not a command", info.Size(), limitInBytes)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// printUsage prints the token/cost summary line; silent without stats.
func printUsage(report rellm.Report) {
	if line := cluesh.UsageSummary(report); line != "" {
		fmt.Println(line)
	}
}

func waitWithProgressbar(ctx context.Context, agent *rellm.Agent, p *rellm.Prompt) (rellm.Report, error) {

	reportChannel := make(chan agentReport)
	go func() {
		report, err := agent.Execute(ctx, p)
		reportChannel <- agentReport{report: report, err: err}
	}()

	// progress marks go to stderr so piped stdout stays clean
	ticker := time.NewTicker(1600 * time.Millisecond)
	defer ticker.Stop()
	defer fmt.Fprintln(os.Stderr)

	for {
		select {
		case ar := <-reportChannel:
			return ar.report, ar.err
		case <-ticker.C:
			fmt.Fprint(os.Stderr, "=")
		}
	}

}

type agentReport struct {
	report rellm.Report
	err    error
}
