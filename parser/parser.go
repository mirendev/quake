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
		parseFileWithBlockNamespace(),
		parseFileWithFileNamespace(),
		parseTaskWithArgsAndDeps(),
		parseTaskWithDepsOnly(),
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
		parseBalancedBraceContent(),
		func(s string) any {
			return s
		},
	)
}

// parseBalancedBraceContent parses content until a matching closing brace, respecting quotes
func parseBalancedBraceContent() p.Rule {
	// First create a reference for recursive parsing
	ref := p.R("balancedContent")

	rule := p.Star(p.Or(
		// Double quoted string - consume entire quoted content including any braces inside
		p.Seq(
			p.S("\""),
			p.Star(p.Or(
				p.S("\\\""),                      // escaped quote
				p.S("\\\\"),                      // escaped backslash
				p.Seq(p.Not(p.S("\"")), p.Any()), // any other char
			)),
			p.S("\""),
		),
		// Single quoted string - consume entire quoted content including any braces inside
		p.Seq(
			p.S("'"),
			p.Star(p.Or(
				p.S("\\'"),                      // escaped quote
				p.S("\\\\"),                     // escaped backslash
				p.Seq(p.Not(p.S("'")), p.Any()), // any other char
			)),
			p.S("'"),
		),
		// Nested braces (outside quotes) - use reference for recursion
		p.Seq(
			p.S("{"),
			ref,
			p.S("}"),
		),
		// Regular character that's not a closing brace (outside quotes)
		p.Seq(p.Not(p.S("}")), p.Any()),
	))

	// Now set the actual rule in the reference to complete the recursion
	ref.Set(rule)

	return rule
}

// Helper function to parse commands from content string
func parseCommands(content string) []Command {
	commands := []Command{}
	for line := range strings.SplitSeq(content, "\n") {
		// Only trim trailing whitespace to preserve indentation
		line = strings.TrimRight(line, " \t\r")
		if line == "" {
			continue
		}

		cmd := Command{Line: line}

		// Handle special prefixes (check the trimmed version for prefixes)
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "@") {
			cmd.Silent = true
			// Preserve indentation when removing the @ prefix
			leadingSpace := line[:len(line)-len(trimmedLine)]
			cmd.Line = leadingSpace + strings.TrimSpace(trimmedLine[1:])
		} else if strings.HasPrefix(trimmedLine, "-") {
			cmd.ContinueOnError = true
			// Preserve indentation when removing the - prefix
			leadingSpace := line[:len(line)-len(trimmedLine)]
			cmd.Line = leadingSpace + strings.TrimSpace(trimmedLine[1:])
		}

		if cmd.Line != "" || line != "" {
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

func parseFileWithFileNamespace() p.Rule {
	return p.Action(
		p.Seq(
			ws(),
			p.S("namespace"),
			requiredSpace(),
			p.Named("namespace", parseWord()),
			ws(),
			p.Not(p.S("{")), // Make sure there's no opening brace (distinguishes from block namespace)
			p.Named("tasks", parseTasks()),
			ws(),
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

func parseFileWithBlockNamespace() p.Rule {
	return p.Action(
		p.Seq(
			ws(),
			p.S("namespace"),
			requiredSpace(),
			p.Named("name", parseWord()),
			ws(),
			p.S("{"), // Require opening brace for block namespace
			p.Named("content", parseRestOfFile()),
			ws(),
			p.EOS(),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			content := v.Get("content").(string)

			// Parse namespace block manually (content already excludes opening brace)
			namespace := parseNamespaceBlockContent(name, content)

			return QuakeFile{
				Tasks:      []Task{},
				Namespaces: []Namespace{namespace},
			}
		},
	)
}

func parseRestOfFile() p.Rule {
	return p.Transform(
		p.Star(p.Any()),
		func(s string) any {
			return s
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

func parseNamespaceBlockContent(name, content string) Namespace {
	content = strings.TrimSpace(content)

	// Find the matching closing brace using quote-aware parsing
	innerContent := extractContentBeforeClosingBrace(content)

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
	tasks := []Task{}
	content = strings.TrimSpace(content)

	if content == "" {
		return tasks
	}

	// Simple parsing for now - look for "task name {" patterns
	lines := strings.Split(content, "\n")
	i := 0

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			i++
			continue
		}

		// Check for task declaration
		if strings.HasPrefix(line, "task ") {
			task, nextIndex := parseTaskFromLines(lines, i)
			if task != nil {
				tasks = append(tasks, *task)
			}
			i = nextIndex
		} else {
			i++
		}
	}

	return tasks
}

func parseTaskFromLines(lines []string, startIndex int) (*Task, int) {
	if startIndex >= len(lines) {
		return nil, startIndex + 1
	}

	line := strings.TrimSpace(lines[startIndex])
	if !strings.HasPrefix(line, "task ") {
		return nil, startIndex + 1
	}

	// Parse task line: "task name {" or "task name(args) {" or "task name => deps {"
	taskPart := strings.TrimPrefix(line, "task ")
	taskPart = strings.TrimSpace(taskPart)

	var taskName string
	var taskArgs []string
	var taskDeps []string

	// Check for dependencies first
	var beforeDeps string
	if strings.Contains(taskPart, "=>") {
		parts := strings.SplitN(taskPart, "=>", 2)
		beforeDeps = strings.TrimSpace(parts[0])
		depPart := parts[1]

		// Extract dependencies
		if idx := strings.Index(depPart, "{"); idx != -1 {
			depString := strings.TrimSpace(depPart[:idx])
			taskDeps = parseArgumentsFromString(depString)
		} else {
			depString := strings.TrimSpace(depPart)
			taskDeps = parseArgumentsFromString(depString)
		}
	} else {
		beforeDeps = taskPart
	}

	// Now parse name and arguments from beforeDeps
	if strings.Contains(beforeDeps, "(") && strings.Contains(beforeDeps, ")") {
		// Task with arguments
		parts := strings.SplitN(beforeDeps, "(", 2)
		taskName = strings.TrimSpace(parts[0])
		argPart := parts[1]
		if idx := strings.Index(argPart, ")"); idx != -1 {
			argString := argPart[:idx]
			taskArgs = parseArgumentsFromString(argString)
		}
	} else {
		// Task without arguments
		if idx := strings.Index(beforeDeps, "{"); idx != -1 {
			taskName = strings.TrimSpace(beforeDeps[:idx])
		} else {
			taskName = strings.TrimSpace(beforeDeps)
		}
	}

	// Find the task body between { and }
	braceCount := 0
	taskBody := []string{}
	i := startIndex

	// Find opening brace
	for i < len(lines) {
		line := lines[i]
		if strings.Contains(line, "{") {
			braceCount = 1
			// Get any content after the opening brace
			if idx := strings.Index(line, "{"); idx != -1 && idx+1 < len(line) {
				remaining := strings.TrimSpace(line[idx+1:])
				if remaining != "" {
					taskBody = append(taskBody, remaining)
				}
			}
			i++
			break
		}
		i++
	}

	// Collect content until closing brace
	for i < len(lines) && braceCount > 0 {
		line := lines[i]

		// Count braces while respecting quoted strings
		newBraceCount := countBracesRespectingQuotes(line, braceCount)

		if newBraceCount > 0 {
			taskBody = append(taskBody, line)
			braceCount = newBraceCount
		} else if newBraceCount == 0 {
			// Found the closing brace - handle any content before it
			closingBraceIndex := findClosingBraceIndex(line)
			if closingBraceIndex > 0 {
				remaining := strings.TrimSpace(line[:closingBraceIndex])
				if remaining != "" {
					taskBody = append(taskBody, remaining)
				}
			}
			break
		}
		i++
	}

	// Parse commands from task body
	bodyContent := strings.Join(taskBody, "\n")
	commands := parseCommands(bodyContent)

	task := &Task{
		Name:         taskName,
		Arguments:    taskArgs,
		Dependencies: taskDeps,
		Commands:     commands,
	}

	return task, i + 1
}

func parseTaskWithArgsAndDeps() p.Rule {
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
			p.S("=>"),
			ws(),
			p.Named("deps", parseDependencyList()),
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
			deps := v.Get("deps").([]string)
			content := v.Get("content").(string)

			commands := parseCommands(content)

			task := Task{
				Name:         name,
				Arguments:    args,
				Dependencies: deps,
				Commands:     commands,
			}

			return QuakeFile{Tasks: []Task{task}}
		},
	)
}

func parseTaskWithDepsOnly() p.Rule {
	return p.Action(
		p.Seq(
			ws(),
			p.S("task"),
			requiredSpace(),
			p.Named("name", parseWord()),
			ws(),
			p.S("=>"),
			ws(),
			p.Named("deps", parseDependencyList()),
			ws(),
			p.S("{"),
			p.Named("content", parseContent()),
			p.S("}"),
			ws(),
			p.EOS(),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			deps := v.Get("deps").([]string)
			content := v.Get("content").(string)

			commands := parseCommands(content)

			task := Task{
				Name:         name,
				Dependencies: deps,
				Commands:     commands,
			}

			return QuakeFile{Tasks: []Task{task}}
		},
	)
}

func parseDependencyList() p.Rule {
	return p.Transform(
		p.Star(p.Seq(
			p.Not(p.S("{")), // not an opening brace
			p.Any(),         // any character
		)),
		func(s string) any {
			return parseArgumentsFromString(s)
		},
	)
}

// countBracesRespectingQuotes counts braces in a line while ignoring braces inside quoted strings
func countBracesRespectingQuotes(line string, currentCount int) int {
	inSingleQuote := false
	inDoubleQuote := false
	braceCount := currentCount

	for i, char := range line {
		switch char {
		case '\'':
			if !inDoubleQuote {
				// Check if it's escaped
				if i > 0 && line[i-1] == '\\' {
					continue
				}
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				// Check if it's escaped
				if i > 0 && line[i-1] == '\\' {
					continue
				}
				inDoubleQuote = !inDoubleQuote
			}
		case '{':
			if !inSingleQuote && !inDoubleQuote {
				braceCount++
			}
		case '}':
			if !inSingleQuote && !inDoubleQuote {
				braceCount--
				if braceCount < 0 {
					return 0 // Found the matching closing brace
				}
			}
		}
	}

	return braceCount
}

// findClosingBraceIndex finds the index of the task's closing brace, respecting quotes
func findClosingBraceIndex(line string) int {
	inSingleQuote := false
	inDoubleQuote := false
	braceCount := 0

	for i, char := range line {
		switch char {
		case '\'':
			if !inDoubleQuote {
				// Check if it's escaped
				if i > 0 && line[i-1] == '\\' {
					continue
				}
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				// Check if it's escaped
				if i > 0 && line[i-1] == '\\' {
					continue
				}
				inDoubleQuote = !inDoubleQuote
			}
		case '{':
			if !inSingleQuote && !inDoubleQuote {
				braceCount++
			}
		case '}':
			if !inSingleQuote && !inDoubleQuote {
				braceCount--
				if braceCount < 0 {
					return i // Found the closing brace
				}
			}
		}
	}

	return len(line) // No closing brace found, return end of line
}

// extractContentBeforeClosingBrace extracts content before the matching closing brace, respecting quotes
func extractContentBeforeClosingBrace(content string) string {
	inSingleQuote := false
	inDoubleQuote := false
	braceCount := 0

	for i, char := range content {
		switch char {
		case '\'':
			if !inDoubleQuote {
				// Check if it's escaped
				if i > 0 && content[i-1] == '\\' {
					continue
				}
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				// Check if it's escaped
				if i > 0 && content[i-1] == '\\' {
					continue
				}
				inDoubleQuote = !inDoubleQuote
			}
		case '{':
			if !inSingleQuote && !inDoubleQuote {
				braceCount++
			}
		case '}':
			if !inSingleQuote && !inDoubleQuote {
				braceCount--
				if braceCount < 0 {
					// Found the matching closing brace
					return content[:i]
				}
			}
		}
	}

	return content // No closing brace found, return all content
}
