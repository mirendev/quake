package parser

import (
	"strings"

	p "github.com/lab47/peggysue"
)

// ParseQuakefile parses a Quakefile string and returns the AST
func ParseQuakefile(input string) (QuakeFile, bool, error) {
	parser := p.New()
	grammar := buildGrammar()
	result, ok, err := parser.Parse(grammar, input)

	if !ok || err != nil {
		return QuakeFile{}, ok, err
	}

	if result == nil {
		return QuakeFile{Tasks: []Task{}}, true, nil
	}

	return result.(QuakeFile), true, nil
}

func buildGrammar() p.Rule {
	return p.Or(
		parseTaskWithArgs(),
		parseTaskNoArgs(),
		parseEmptyFile(),
	)
}

func parseEmptyFile() p.Rule {
	return p.Action(
		p.Seq(
			ws(),
			p.EOS(),
		),
		func(v p.Values) any {
			return QuakeFile{Tasks: []Task{}}
		},
	)
}

func parseTaskNoArgs() p.Rule {
	return p.Action(
		p.Seq(
			ws(),
			p.S("task"),
			requiredSpace(),
			p.Named("name", parseWord()),
			ws(),
			p.S("{"),
			p.Named("content", parseContent()),
			p.S("}"),
			ws(),
			p.EOS(),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			content := v.Get("content").(string)

			commands := parseCommands(content)

			task := Task{
				Name:     name,
				Commands: commands,
			}

			return QuakeFile{Tasks: []Task{task}}
		},
	)
}

func parseTaskWithArgs() p.Rule {
	return p.Action(
		p.Seq(
			ws(),
			p.S("task"),
			requiredSpace(),
			p.Named("name", parseWord()),
			p.S("("),
			p.Named("args", parseArgList()),
			p.S(")"),
			ws(),
			p.S("{"),
			p.Named("content", parseContent()),
			p.S("}"),
			ws(),
			p.EOS(),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			args := v.Get("args").([]string)
			content := v.Get("content").(string)

			commands := parseCommands(content)

			task := Task{
				Name:      name,
				Arguments: args,
				Commands:  commands,
			}

			return QuakeFile{Tasks: []Task{task}}
		},
	)
}

func parseArgList() p.Rule {
	return p.Transform(
		p.Star(p.Seq(
			p.Not(p.S(")")), // not a closing paren
			p.Any(),         // any character
		)),
		func(s string) any {
			return parseArgumentsFromString(s)
		},
	)
}

// ws matches optional whitespace (spaces, tabs, newlines)
func ws() p.Rule {
	return p.Star(p.Or(
		p.S(" "),
		p.S("\t"),
		p.S("\n"),
		p.S("\r"),
	))
}

// requiredSpace matches one or more spaces/tabs (not newlines)
func requiredSpace() p.Rule {
	return p.Plus(p.Or(
		p.S(" "),
		p.S("\t"),
	))
}

func parseWord() p.Rule {
	return p.Transform(
		p.Plus(p.Or(
			p.Range('a', 'z'),
			p.Range('A', 'Z'),
			p.Range('0', '9'),
			p.S("_"),
		)),
		func(s string) any {
			return s
		},
	)
}

func parseContent() p.Rule {
	return p.Transform(
		p.Star(p.Seq(
			p.Not(p.S("}")),
			p.Any(),
		)),
		func(s string) any {
			return s
		},
	)
}

// Helper function to parse commands from content string
func parseCommands(content string) []Command {
	commands := []Command{}
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		cmd := Command{Line: line}

		// Handle special prefixes
		if strings.HasPrefix(line, "@") {
			cmd.Silent = true
			cmd.Line = strings.TrimSpace(line[1:])
		} else if strings.HasPrefix(line, "-") {
			cmd.ContinueOnError = true
			cmd.Line = strings.TrimSpace(line[1:])
		}

		if cmd.Line != "" {
			commands = append(commands, cmd)
		}
	}
	return commands
}

// For now, we'll use a simpler approach to parse argument lists
// We'll enhance parseArgList to handle the string directly
func parseArgumentsFromString(argString string) []string {
	if strings.TrimSpace(argString) == "" {
		return []string{}
	}

	args := []string{}
	parts := strings.Split(argString, ",")
	for _, part := range parts {
		arg := strings.TrimSpace(part)
		if arg != "" {
			args = append(args, arg)
		}
	}
	return args
}

