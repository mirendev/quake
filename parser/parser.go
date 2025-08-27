package parser

import (
	"strings"

	p "github.com/lab47/peggysue"
)

// Grammar holds all the parsing rules
type Grammar struct {
	quakeFile              p.Rule
	topLevelElement        p.Rule
	comment                p.Rule
	fileNamespaceDirective p.Rule
	variable               p.Rule
	multilineStringVar     p.Rule
	simpleVariable         p.Rule
	variableValue          p.Rule
	commandSubstitution    p.Rule
	expressionValue        p.Rule
	quotedString           p.Rule
	task                   p.Rule
	taskSimple             p.Rule
	taskWithArgs           p.Rule
	taskWithDeps           p.Rule
	taskWithArgsAndDeps    p.Rule
	namespace              p.Rule
	namespaceRef           p.Rule
	argList                p.Rule
	dependencies           p.Rule
	word                   p.Rule
	ws                     p.Rule
	requiredSpace          p.Rule
	content                p.Rule
	balancedBraceContent   p.Rule
	// Command parsing rules
	commandLine       p.Rule
	commandElement    p.Rule
	commandElements   p.Rule
	plainText         p.Rule
	backtickCmd       p.Rule
	variableRef       p.Rule
	expressionElement p.Rule
}

// NewGrammar creates and initializes a new grammar
func NewGrammar() *Grammar {
	g := &Grammar{}
	g.init()
	return g
}

// init initializes all the grammar rules
func (g *Grammar) init() {
	// Create references first
	namespaceRef := p.R("namespace")
	g.namespaceRef = namespaceRef
	balancedRef := p.R("balancedContent")

	// Define basic rules
	g.ws = p.Star(p.Or(
		p.S(" "),
		p.S("\t"),
		p.S("\n"),
		p.S("\r"),
	))

	g.requiredSpace = p.Plus(p.Or(
		p.S(" "),
		p.S("\t"),
	))

	g.word = p.Transform(
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

	// Define content parsing with balanced braces
	balancedRule := p.Star(p.Or(
		// Double quoted string
		p.Seq(
			p.S("\""),
			p.Star(p.Or(
				p.S("\\\""),
				p.S("\\\\"),
				p.Seq(p.Not(p.S("\"")), p.Any()),
			)),
			p.S("\""),
		),
		// Single quoted string
		p.Seq(
			p.S("'"),
			p.Star(p.Or(
				p.S("\\'"),
				p.S("\\\\"),
				p.Seq(p.Not(p.S("'")), p.Any()),
			)),
			p.S("'"),
		),
		// Nested braces
		p.Seq(
			p.S("{"),
			balancedRef,
			p.S("}"),
		),
		// Regular character that's not a closing brace
		p.Seq(p.Not(p.S("}")), p.Any()),
	))
	balancedRef.Set(balancedRule)
	g.balancedBraceContent = balancedRule

	g.content = p.Transform(
		g.balancedBraceContent,
		func(s string) any {
			return s
		},
	)

	// Define comment
	g.comment = p.Action(
		p.Seq(
			p.S("#"),
			p.Star(p.Seq(p.Not(p.S("\n")), p.Any())),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			return nil // Comments are ignored
		},
	)

	// Define file namespace directive (unquoted, no opening brace)
	g.fileNamespaceDirective = p.Action(
		p.Seq(
			p.S("namespace"),
			g.requiredSpace,
			p.Named("name", g.word),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			return FileNamespaceDirective{Name: v.Get("name").(string)}
		},
	)

	// Define variable parsing rules
	g.quotedString = p.Transform(
		p.Seq(
			p.S("\""),
			p.Star(p.Or(
				p.S("\\\""),
				p.S("\\\\"),
				p.Seq(p.Not(p.S("\"")), p.Any()),
			)),
			p.S("\""),
		),
		func(s string) any { return s },
	)

	g.commandSubstitution = p.Action(
		p.Seq(
			p.S("`"),
			p.Named("cmd", p.Transform(
				p.Star(p.Seq(p.Not(p.S("`")), p.Any())),
				func(s string) any { return s },
			)),
			p.S("`"),
		),
		func(v p.Values) any {
			return Variable{
				Value:               "`" + v.Get("cmd").(string) + "`",
				CommandSubstitution: true,
			}
		},
	)

	g.expressionValue = p.Action(
		p.Seq(
			p.S("{{"),
			p.Named("expr", p.Transform(
				p.Star(p.Seq(p.Not(p.S("}}")), p.Any())),
				func(s string) any { return s },
			)),
			p.S("}}"),
		),
		func(v p.Values) any {
			return Variable{
				Value:        "{{" + v.Get("expr").(string) + "}}",
				IsExpression: true,
			}
		},
	)

	g.variableValue = p.Or(
		g.commandSubstitution,
		g.expressionValue,
		g.quotedString,
	)

	g.multilineStringVar = p.Action(
		p.Seq(
			p.Named("name", g.word),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.S("="),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.S("\"\"\""),
			p.Or(p.S("\n"), p.EOS()),
			p.Named("content", p.Transform(
				p.Star(p.Seq(
					p.Not(p.S("\"\"\"")),
					p.Any(),
				)),
				func(s string) any { return s },
			)),
			p.S("\"\"\""),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			return Variable{
				Name:        v.Get("name").(string),
				Value:       v.Get("content").(string),
				IsMultiline: true,
			}
		},
	)

	g.simpleVariable = p.Action(
		p.Seq(
			p.Named("name", g.word),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.S("="),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Named("value", g.variableValue),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			value := v.Get("value")
			switch val := value.(type) {
			case Variable:
				val.Name = v.Get("name").(string)
				return val
			default:
				return Variable{
					Name:  v.Get("name").(string),
					Value: val.(string),
				}
			}
		},
	)

	g.variable = p.Or(
		g.multilineStringVar,
		g.simpleVariable,
	)

	// Define argument and dependency parsing
	g.argList = p.Transform(
		p.Star(p.Seq(
			p.Not(p.S(")")),
			p.Any(),
		)),
		func(s string) any {
			return parseArgumentsFromString(s)
		},
	)

	g.dependencies = p.Transform(
		p.Star(p.Seq(
			p.Not(p.S("{")),
			p.Any(),
		)),
		func(s string) any {
			return parseDependenciesFromString(s)
		},
	)

	// Define task parsing rules
	g.taskSimple = p.Action(
		p.Seq(
			p.S("task"),
			g.requiredSpace,
			p.Named("name", g.word),
			g.ws,
			p.S("{"),
			p.Named("content", g.content),
			p.S("}"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			content := v.Get("content").(string)
			commands := parseCommands(content)

			return Task{
				Name:     name,
				Commands: commands,
			}
		},
	)

	g.taskWithArgs = p.Action(
		p.Seq(
			p.S("task"),
			g.requiredSpace,
			p.Named("name", g.word),
			p.S("("),
			p.Named("args", g.argList),
			p.S(")"),
			g.ws,
			p.S("{"),
			p.Named("content", g.content),
			p.S("}"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			args := v.Get("args").([]string)
			content := v.Get("content").(string)
			commands := parseCommands(content)

			return Task{
				Name:      name,
				Arguments: args,
				Commands:  commands,
			}
		},
	)

	g.taskWithDeps = p.Action(
		p.Seq(
			p.S("task"),
			g.requiredSpace,
			p.Named("name", g.word),
			g.ws,
			p.S("=>"),
			g.ws,
			p.Named("deps", g.dependencies),
			g.ws,
			p.S("{"),
			p.Named("content", g.content),
			p.S("}"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			deps := v.Get("deps").([]string)
			content := v.Get("content").(string)
			commands := parseCommands(content)

			return Task{
				Name:         name,
				Dependencies: deps,
				Commands:     commands,
			}
		},
	)

	g.taskWithArgsAndDeps = p.Action(
		p.Seq(
			p.S("task"),
			g.requiredSpace,
			p.Named("name", g.word),
			p.S("("),
			p.Named("args", g.argList),
			p.S(")"),
			g.ws,
			p.S("=>"),
			g.ws,
			p.Named("deps", g.dependencies),
			g.ws,
			p.S("{"),
			p.Named("content", g.content),
			p.S("}"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			args := v.Get("args").([]string)
			deps := v.Get("deps").([]string)
			content := v.Get("content").(string)
			commands := parseCommands(content)

			return Task{
				Name:         name,
				Arguments:    args,
				Dependencies: deps,
				Commands:     commands,
			}
		},
	)

	g.task = p.Or(
		g.taskWithArgsAndDeps,
		g.taskWithDeps,
		g.taskWithArgs,
		g.taskSimple,
	)

	// Define namespace rule
	namespaceRule := p.Action(
		p.Seq(
			p.S("namespace"),
			g.requiredSpace,
			p.Named("name", g.word),
			g.ws,
			p.S("{"),
			p.Named("elements", p.Many(p.Action(
				p.Seq(
					g.ws,
					p.Named("element", p.Or(
						g.comment,
						g.variable,
						g.task,
						g.namespaceRef,
					)),
				),
				func(v p.Values) any {
					return v.Get("element")
				},
			), 0, -1, func(values []any) any {
				return values
			})),
			p.S("}"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			ns := Namespace{
				Name:       name,
				Tasks:      []Task{},
				Variables:  []Variable{},
				Namespaces: []Namespace{},
			}

			elements := v.Get("elements")
			if elements != nil {
				for _, elem := range elements.([]any) {
					if elem == nil {
						continue
					}
					switch e := elem.(type) {
					case Task:
						ns.Tasks = append(ns.Tasks, e)
					case Variable:
						ns.Variables = append(ns.Variables, e)
					case Namespace:
						ns.Namespaces = append(ns.Namespaces, e)
					}
				}
			}

			return ns
		},
	)
	// Set the reference using type assertion
	if ref, ok := namespaceRef.(interface{ Set(p.Rule) }); ok {
		ref.Set(namespaceRule)
	}
	g.namespace = namespaceRule

	// Define top-level element
	g.topLevelElement = p.Action(
		p.Seq(
			g.ws,
			p.Named("element", p.Or(
				g.comment,
				g.fileNamespaceDirective,
				g.variable,
				g.task,
				g.namespace,
			)),
		),
		func(v p.Values) any {
			return v.Get("element")
		},
	)

	// Define the main Quakefile rule
	g.quakeFile = p.Action(
		p.Seq(
			p.Named("elements", p.Many(g.topLevelElement, 0, -1, func(values []any) any {
				return values
			})),
			g.ws,
			p.EOS(),
		),
		func(v p.Values) any {
			qf := QuakeFile{
				Tasks:      []Task{},
				Namespaces: []Namespace{},
				Variables:  []Variable{},
			}

			elements := v.Get("elements")
			// Debug: check what type elements is
			if elements != nil {
				// Try to handle it as a slice
				switch elems := elements.(type) {
				case []any:
					for _, elem := range elems {
						if elem == nil {
							continue
						}
						switch e := elem.(type) {
						case Task:
							qf.Tasks = append(qf.Tasks, e)
						case Namespace:
							qf.Namespaces = append(qf.Namespaces, e)
						case Variable:
							qf.Variables = append(qf.Variables, e)
						case FileNamespaceDirective:
							qf.FileNamespace = e.Name
						}
					}
				default:
					// Single element?
					switch e := elements.(type) {
					case Task:
						qf.Tasks = append(qf.Tasks, e)
					case Namespace:
						qf.Namespaces = append(qf.Namespaces, e)
					case Variable:
						qf.Variables = append(qf.Variables, e)
					case FileNamespaceDirective:
						qf.FileNamespace = e.Name
					}
				}
			}

			return qf
		},
	)

	// Define command parsing rules
	// Variable reference: $NAME
	g.variableRef = p.Action(
		p.Seq(
			p.S("$"),
			p.Named("name", p.Transform(
				p.Plus(p.Or(
					p.Range('a', 'z'),
					p.Range('A', 'Z'),
					p.Range('0', '9'),
					p.S("_"),
				)),
				func(s string) any { return s },
			)),
		),
		func(v p.Values) any {
			return VariableElement{Name: v.Get("name").(string)}
		},
	)

	// Expression: {{expr}}
	g.expressionElement = p.Action(
		p.Seq(
			p.S("{{"),
			p.Named("expr", p.Transform(
				p.Star(p.Seq(
					p.Not(p.S("}}")),
					p.Any(),
				)),
				func(s string) any { return s },
			)),
			p.S("}}"),
		),
		func(v p.Values) any {
			return ExpressionElement{Expression: v.Get("expr").(string)}
		},
	)

	// Backtick command: `cmd`
	g.backtickCmd = p.Action(
		p.Seq(
			p.S("`"),
			p.Named("cmd", p.Transform(
				p.Star(p.Seq(
					p.Not(p.S("`")),
					p.Any(),
				)),
				func(s string) any { return s },
			)),
			p.S("`"),
		),
		func(v p.Values) any {
			return BacktickElement{Command: v.Get("cmd").(string)}
		},
	)

	// Plain text that's not a special element
	g.plainText = p.Transform(
		p.Plus(p.Seq(
			p.Not(p.Or(
				p.S("$"),
				p.S("{{"),
				p.S("`"),
				p.S("\n"),
				p.EOS(),
			)),
			p.Any(),
		)),
		func(s string) any {
			return StringElement{Value: s}
		},
	)

	// A single command element
	g.commandElement = p.Or(
		g.expressionElement,
		g.backtickCmd,
		g.variableRef,
		g.plainText,
	)

	// Command elements (multiple elements)
	g.commandElements = p.Many(g.commandElement, 0, -1, func(values []any) any {
		elements := make([]CommandElement, 0, len(values))
		for _, v := range values {
			if elem, ok := v.(CommandElement); ok {
				elements = append(elements, elem)
			}
		}
		return elements
	})

	// A complete command line
	g.commandLine = p.Action(
		p.Seq(
			p.Named("elements", g.commandElements),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			elements := v.Get("elements").([]CommandElement)
			return Command{Elements: elements}
		},
	)
}

// ParseQuakefile parses a Quakefile string and returns the AST
func ParseQuakefile(input string) (QuakeFile, bool, error) {
	parser := p.New()
	grammar := NewGrammar()
	result, ok, err := parser.Parse(grammar.quakeFile, input, p.WithErrors())

	if !ok || err != nil {
		return QuakeFile{}, ok, err
	}

	if result == nil {
		return QuakeFile{Tasks: []Task{}}, true, nil
	}

	return result.(QuakeFile), true, nil
}

// FileNamespaceDirective represents a file-level namespace directive
type FileNamespaceDirective struct {
	Name string
}

// Helper function to parse commands from content string
func parseCommands(content string) []Command {
	// Create a parser with the command line grammar
	parser := p.New()
	grammar := NewGrammar()

	commands := []Command{}
	for line := range strings.SplitSeq(content, "\n") {
		// Only trim trailing whitespace to preserve indentation
		line = strings.TrimRight(line, " \t\r")
		if line == "" {
			continue
		}

		// Check for special prefixes
		trimmedLine := strings.TrimSpace(line)
		silent := false
		continueOnError := false

		// Handle special prefixes
		if strings.HasPrefix(trimmedLine, "@") {
			silent = true
			trimmedLine = strings.TrimSpace(trimmedLine[1:])
		} else if strings.HasPrefix(trimmedLine, "-") {
			continueOnError = true
			trimmedLine = strings.TrimSpace(trimmedLine[1:])
		}

		// Parse the command line using PEG grammar
		result, ok, _ := parser.Parse(grammar.commandElements, trimmedLine, p.WithErrors())

		var elements []CommandElement
		if ok && result != nil {
			if elems, ok := result.([]CommandElement); ok {
				elements = elems
			} else {
				// Fallback to simple string if parsing fails
				elements = []CommandElement{StringElement{Value: trimmedLine}}
			}
		} else {
			// If parsing fails, treat the whole line as a string
			elements = []CommandElement{StringElement{Value: trimmedLine}}
		}

		cmd := Command{
			Elements:        elements,
			Silent:          silent,
			ContinueOnError: continueOnError,
		}
		commands = append(commands, cmd)
	}
	return commands
}

// parseArgumentsFromString parses argument string into array
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

// parseDependenciesFromString parses dependency string into array
func parseDependenciesFromString(depString string) []string {
	depString = strings.TrimSpace(depString)
	if depString == "" {
		return []string{}
	}

	deps := []string{}
	parts := strings.FieldsFunc(depString, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})

	for _, part := range parts {
		if part != "" {
			deps = append(deps, part)
		}
	}
	return deps
}

// Legacy functions kept for compatibility
func parseFileWithFileNamespace() p.Rule {
	return p.Action(
		p.Seq(
			p.Star(p.Or(p.S(" "), p.S("\t"), p.S("\n"), p.S("\r"))),
			p.S("namespace"),
			p.Plus(p.Or(p.S(" "), p.S("\t"))),
			p.Named("namespace", p.Transform(
				p.Plus(p.Or(
					p.Range('a', 'z'),
					p.Range('A', 'Z'),
					p.Range('0', '9'),
					p.S("_"),
				)),
				func(s string) any { return s },
			)),
			p.Star(p.Or(p.S(" "), p.S("\t"), p.S("\n"), p.S("\r"))),
			p.Not(p.S("{")),
			p.Named("tasks", parseTasks()),
			p.Star(p.Or(p.S(" "), p.S("\t"), p.S("\n"), p.S("\r"))),
			p.EOS(),
		),
		func(v p.Values) any {
			namespace := v.Get("namespace").(string)
			tasks := v.Get("tasks").([]Task)

			return QuakeFile{
				FileNamespace: namespace,
				Tasks:         tasks,
			}
		},
	)
}

func parseNamespaceFromContent(name, content string) Namespace {
	content = strings.TrimSpace(content)

	// Must start with {
	if !strings.HasPrefix(content, "{") {
		return Namespace{Name: name}
	}

	// Find the matching closing brace
	braceCount := 0
	var innerContent string

	for i, char := range content {
		switch char {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				// Found the matching closing brace
				innerContent = content[1:i] // Skip opening and closing braces
				break
			}
		}
	}

	tasks := parseTasksFromContent(innerContent)

	return Namespace{
		Name:  name,
		Tasks: tasks,
	}
}

func parseTasks() p.Rule {
	return p.Transform(
		p.Star(p.Any()),
		func(s string) any {
			return parseTasksFromContent(s)
		},
	)
}

func parseTasksFromContent(content string) []Task {
	// This is a temporary implementation
	// In a proper implementation, we'd use the PEG parser recursively
	return []Task{}
}

// parseNamespaceFromLines parses a namespace from lines
func parseNamespaceFromLines(lines []string, startIndex int) (*Namespace, int) {
	// This is a temporary implementation
	// In a proper implementation, we'd use the PEG parser recursively
	return nil, startIndex + 1
}
