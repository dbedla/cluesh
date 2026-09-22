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

func printTest() {
	r := rellm.Report{Message: rawjs}
	cmd, err := cluesh.ParseAgentResult(r)
	if err != nil {
		panic(err)
	}

	cluesh.Print(&cluesh.DarkMode{}, cmd)
}

const rawjs = `
 {
  "finalCommand": "find . -name '*.go' -exec wc -l {} + | awk '$1 > 10 && $2 != \"total\" {print $2}'",
  "subCommands": [
    {
      "command": "find . -name '*.go' -exec wc -l {} +",
      "arguments": [
        {
          "argument": ".",
          "description": "start searching in the current directory (recursively)"
        },
        {
          "argument": "-name '*.go'",
          "description": "match only files whose name ends with .go"
        },
        {
          "argument": "-exec wc -l {} +",
          "description": "run wc -l on all found files, batching them into as few invocations as possible; wc -l prints the line count followed by the filename for each file (plus a 'total' line when multiple files are given)"
        }
      ]
    },
    {
      "command": "awk '$1 > 10 && $2 != \"total\" {print $2}'",
      "arguments": [
        {
          "argument": "$1 > 10",
          "description": "condition: first field (line count reported by wc) must be greater than 10"
        },
        {
          "argument": "$2 != \"total\"",
          "description": "condition: skip the summary line that wc prints when multiple files are batched"
        },
        {
          "argument": "{print $2}",
          "description": "print only the second field, i.e. the filename, for lines matching both conditions"
        }
      ]
    }
  ]
, "notes": "If filenames may contain spaces, replace the pipeline with: find . -name '*.go' -exec awk 'END {if (NR > 10) print FILENAME}' {} \\; — but for typical Go projects the pipeline above is fast and sufficient."  , "redOnly": true}
`
