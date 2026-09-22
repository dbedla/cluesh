package cluesh

import (
	"github.com/dbedla/rellm/pkg/rellm"
)

const rawjson = `
{
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
  ]
  ,
  "notes": "The find command recursively lists every regular file with a .go extension. xargs -n1 feeds one file at a time to wc -l, which avoids the 'total' summary line that multi-file wc output would produce. awk '$1 > 10' filters out files with 10 or fewer lines, leaving one output line per matching file in the format '<line_count> <filename>'."
  ,
  "read_only": true
  }
`

func TPrint() {

	report := rellm.Report{Message: rawjson}

	cmd, err := ParseAgentResult(report)
	if err != nil {
		panic(err)
	}

	Print(LightMode{}, cmd)
}
