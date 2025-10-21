package evaluator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"miren.dev/quake/internal/color"
	"miren.dev/quake/parser"
)

// Evaluator handles task execution
type Evaluator struct {
	quakefile *parser.QuakeFile
	env       map[string]string
	taskArgs  []string // Arguments passed to the current task
}

// New creates a new evaluator
func New(quakefile *parser.QuakeFile) *Evaluator {
	e := &Evaluator{
		quakefile: quakefile,
		env:       make(map[string]string),
	}
	// Load global variables into the environment
	e.loadGlobalVariables()
	return e
}

// loadGlobalVariables loads top-level variables from the Quakefile into the environment
func (e *Evaluator) loadGlobalVariables() {
	for _, variable := range e.quakefile.Variables {
		value := e.evaluateVariable(variable)
		e.env[variable.Name] = value
	}
}

// evaluateVariable evaluates a variable's value based on its type
func (e *Evaluator) evaluateVariable(variable parser.Variable) string {
	// Handle command substitution (backticks)
	if variable.CommandSubstitution {
		if cmdStr, ok := variable.Value.(string); ok {
			// Remove the backticks from the command string
			cmdStr = strings.Trim(cmdStr, "`")
			// Execute the command and capture output
			cmd := exec.Command("sh", "-c", cmdStr)
			output, err := cmd.Output()
			if err != nil {
				// If command fails, return empty string
				return ""
			}
			// Trim whitespace from output
			return strings.TrimSpace(string(output))
		}
		return ""
	}

	// Handle expressions ({{...}})
	if variable.IsExpression {
		if expr, ok := variable.Value.(parser.Expression); ok {
			return e.expressionToString(expr)
		}
		return ""
	}

	// Handle plain string values
	if str, ok := variable.Value.(string); ok {
		// Check if it's a quoted string and unquote it
		str = strings.TrimSpace(str)
		if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
			// Remove surrounding quotes
			str = str[1 : len(str)-1]
			// Unescape common escape sequences
			str = strings.ReplaceAll(str, "\\\"", "\"")
			str = strings.ReplaceAll(str, "\\\\", "\\")
			str = strings.ReplaceAll(str, "\\n", "\n")
			str = strings.ReplaceAll(str, "\\t", "\t")
		}
		// Expand any variable references within the string value
		return e.expandShellVariables(str)
	}

	return ""
}

// RunTask executes a specific task by name (without arguments)
func (e *Evaluator) RunTask(taskName string) error {
	return e.RunTaskWithArgs(taskName, nil)
}

// RunTaskWithArgs executes a specific task by name with arguments
func (e *Evaluator) RunTaskWithArgs(taskName string, args []string) error {
	// Handle default task if no name provided
	if taskName == "" {
		taskName = "default"
	}

	// Find the task
	task := e.findTask(taskName)
	if task == nil {
		return fmt.Errorf("task '%s' not found", taskName)
	}

	// Note: We allow fewer arguments than defined - they'll just be empty strings
	// This allows for optional arguments with default values using || in expressions

	// Save current args and restore after task execution
	oldArgs := e.taskArgs
	e.taskArgs = args
	defer func() { e.taskArgs = oldArgs }()

	// Set up argument variables
	for i, argName := range task.Arguments {
		if i < len(args) {
			e.env[argName] = args[i]
		} else {
			e.env[argName] = ""
		}
	}

	// Execute dependencies first (without arguments)
	for _, dep := range task.Dependencies {
		if err := e.RunTask(dep); err != nil {
			return fmt.Errorf("dependency '%s' failed: %w", dep, err)
		}
	}

	// Execute the task
	if len(args) > 0 {
		fmt.Printf("%s [ %s %s ]\n", color.FaintText("┌────"), color.BoldText(taskName), strings.Join(args, ", "))
	} else {
		fmt.Printf("%s [ %s ]\n", color.FaintText("┌────"), color.BoldText(taskName))
	}
	return e.executeTask(task)
}

// findTask locates a task by name, checking namespaces if needed
func (e *Evaluator) findTask(name string) *parser.Task {
	// First, look in top-level tasks (including flattened namespace:name tasks)
	for i := range e.quakefile.Tasks {
		if e.quakefile.Tasks[i].Name == name {
			return &e.quakefile.Tasks[i]
		}
	}

	// If not found and contains ':', also check actual namespace structures
	if strings.Contains(name, ":") {
		parts := strings.Split(name, ":")
		return e.findNamespacedTask(parts, e.quakefile.Namespaces)
	}

	return nil
}

// findNamespacedTask searches for a task in namespaces
func (e *Evaluator) findNamespacedTask(parts []string, namespaces []parser.Namespace) *parser.Task {
	if len(parts) == 0 {
		return nil
	}

	// Look for matching namespace
	for _, ns := range namespaces {
		if ns.Name == parts[0] {
			if len(parts) == 2 {
				// Look for task in this namespace
				for i := range ns.Tasks {
					if ns.Tasks[i].Name == parts[1] {
						return &ns.Tasks[i]
					}
				}
			} else if len(parts) > 2 {
				// Recurse into nested namespaces
				return e.findNamespacedTask(parts[1:], ns.Namespaces)
			}
		}
	}

	return nil
}

// executeTask runs all commands in a task
func (e *Evaluator) executeTask(task *parser.Task) error {
	// Handle Go tasks differently
	if task.IsGoTask {
		return e.executeGoTask(task)
	}

	for i, cmd := range task.Commands {
		isLastCommand := i == len(task.Commands)-1
		if err := e.executeCommandWithPosition(cmd, isLastCommand); err != nil {
			if !cmd.ContinueOnError {
				return err
			}
			// Continue on error if specified
			fmt.Printf("Warning: command failed but continuing: %v\n", err)
		}
	}
	return nil
}

// executeGoTask runs a Go task by invoking go run with the dispatcher
func (e *Evaluator) executeGoTask(task *parser.Task) error {
	if task.GoDispatcher == "" {
		return fmt.Errorf("Go task '%s' has no dispatcher", task.Name)
	}

	if task.GoSourceDir == "" {
		return fmt.Errorf("Go task '%s' has no source directory", task.Name)
	}

	// Build command arguments: go run <dir> taskname args...
	// This will compile all .go files in the directory together
	// Use absolute path to the Go source directory
	qtasksPath, _ := filepath.Abs(task.GoSourceDir)
	args := []string{"run", qtasksPath, task.Name}
	args = append(args, e.taskArgs...)

	// Execute using go run from the project root
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Go task failed: %w", err)
	}

	return nil
}

// executeCommand runs a single command (for backward compatibility)
func (e *Evaluator) executeCommand(cmd parser.Command) error {
	return e.executeCommandWithPosition(cmd, true)
}

// executeCommandWithPosition runs a single command with position info
func (e *Evaluator) executeCommandWithPosition(cmd parser.Command, isLast bool) error {
	// Check if this is an @echo command - use native printer instead of shell
	if cmd.Silent && e.isEchoCommand(cmd) {
		return e.executeNativeEcho(cmd)
	}

	// Convert command to string
	cmdStr := e.commandToString(cmd)

	// Handle silent mode
	if cmd.Silent {
		// Don't print the command
	} else {
		prefix := "├"
		if isLast {
			prefix = "└"
		}
		fmt.Printf("%s %s\n", color.FaintText(prefix), cmdStr)
	}

	// Execute via shell
	shellCmd := exec.Command("sh", "-c", cmdStr)
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr
	shellCmd.Stdin = os.Stdin

	err := shellCmd.Run()
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// isEchoCommand checks if a command is an echo command
func (e *Evaluator) isEchoCommand(cmd parser.Command) bool {
	if len(cmd.Elements) == 0 {
		return false
	}

	// Check if the first element is "echo"
	first := cmd.Elements[0]
	if str, ok := first.(parser.StringElement); ok {
		trimmed := strings.TrimSpace(str.Value)
		// Check if it's exactly "echo" or starts with "echo "
		return trimmed == "echo" || strings.HasPrefix(trimmed, "echo ")
	}

	return false
}

// executeNativeEcho executes an echo command using native Go printing
func (e *Evaluator) executeNativeEcho(cmd parser.Command) error {
	if len(cmd.Elements) == 0 {
		fmt.Printf("%s\n", color.FaintText("│"))
		return nil
	}

	var output strings.Builder

	for i, elem := range cmd.Elements {
		switch el := elem.(type) {
		case parser.StringElement:
			val := el.Value
			// Skip "echo " prefix from the first element
			if i == 0 {
				trimmed := strings.TrimSpace(val)
				if trimmed == "echo" {
					continue
				} else if strings.HasPrefix(trimmed, "echo ") {
					// Remove "echo " prefix
					val = strings.TrimPrefix(val, "echo ")
					val = strings.TrimLeft(val, " \t")
				}
			}

			// Handle quotes - strip leading/trailing quotes but preserve content
			val = e.stripQuotesForEcho(val, i == 1 || (i == 0 && !strings.HasPrefix(strings.TrimSpace(el.Value), "echo")))
			output.WriteString(val)
		case parser.VariableElement:
			// Resolve variable
			if val, ok := e.env[el.Name]; ok {
				output.WriteString(val)
			} else if val, ok := os.LookupEnv(el.Name); ok {
				output.WriteString(val)
			}
			// If variable not found, don't output anything (bash behavior)
		case parser.BacktickElement:
			// For native echo, we could execute the backtick command
			// but for simplicity, we'll fall back to the full command string
			// This is an edge case that's less common with @echo
			cmdStr := e.commandToString(cmd)
			// Remove the "echo " prefix
			cmdStr = strings.TrimSpace(strings.TrimPrefix(cmdStr, "echo"))
			fmt.Printf("%s %s\n", color.FaintText("│"), cmdStr)
			return nil
		case parser.ExpressionElement:
			// Evaluate the expression
			val := e.expressionToString(el.Expression)
			output.WriteString(val)
		}
	}

	// Print with colored pipe prefix
	fmt.Printf("%s %s\n", color.FaintText("│"), output.String())
	return nil
}

// stripQuotesForEcho removes quotes and expands variables for echo command
// It handles multiple quoted sections within a single string
func (e *Evaluator) stripQuotesForEcho(s string, isFirstArg bool) string {
	// Don't trim! We need to preserve spaces inside quotes
	// If the entire string is a single quoted section, handle it simply
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' && !strings.Contains(s[1:len(s)-1], "\"") {
		return e.expandShellVariables(s[1 : len(s)-1])
	} else if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' && !strings.Contains(s[1:len(s)-1], "'") {
		// Single quotes - no variable expansion
		return s[1 : len(s)-1]
	}

	// Handle strings with multiple quoted sections or mixed quoted/unquoted parts
	// Process character by character, removing quotes and expanding variables in double-quoted sections
	var result strings.Builder
	inDoubleQuote := false
	inSingleQuote := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if ch == '"' && !inSingleQuote {
			// Toggle double quote state, but don't include the quote character
			inDoubleQuote = !inDoubleQuote
			continue
		} else if ch == '\'' && !inDoubleQuote {
			// Toggle single quote state, but don't include the quote character
			inSingleQuote = !inSingleQuote
			continue
		}

		result.WriteByte(ch)
	}

	// Expand variables in the result (respecting the quoting context)
	return e.expandShellVariables(result.String())
}

// unquoteString removes surrounding quotes and expands shell variables
func (e *Evaluator) unquoteString(s string) string {
	s = strings.TrimSpace(s)

	// Remove surrounding double quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
		// Expand variables in double-quoted strings
		s = e.expandShellVariables(s)
	} else if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		// Remove surrounding single quotes (no variable expansion in single quotes)
		s = s[1 : len(s)-1]
	} else {
		// No quotes, still expand variables
		s = e.expandShellVariables(s)
	}

	return s
}

// expandShellVariables expands ${VAR} and $VAR syntax
func (e *Evaluator) expandShellVariables(s string) string {
	// Expand ${VAR} syntax
	result := os.Expand(s, func(key string) string {
		// Check evaluator environment first
		if val, ok := e.env[key]; ok {
			return val
		}
		// Fall back to system environment
		return os.Getenv(key)
	})

	return result
}

// commandToString converts a command to an executable string
func (e *Evaluator) commandToString(cmd parser.Command) string {
	var parts []string

	for _, elem := range cmd.Elements {
		switch el := elem.(type) {
		case parser.StringElement:
			parts = append(parts, el.Value)
		case parser.VariableElement:
			// For now, use environment variable or empty string
			if val, ok := e.env[el.Name]; ok {
				parts = append(parts, val)
			} else if val, ok := os.LookupEnv(el.Name); ok {
				parts = append(parts, val)
			} else {
				// If we don't have it, just include as-is (shell will evaluate)
				parts = append(parts, "$"+el.Name)
			}
		case parser.BacktickElement:
			// For now, include the backtick command as-is (shell will evaluate)
			parts = append(parts, "`"+el.Command+"`")
		case parser.ExpressionElement:
			// For now, convert expression to string representation
			parts = append(parts, e.expressionToString(el.Expression))
		default:
			// Unknown element type, skip
		}
	}

	return strings.Join(parts, "")
}

// expressionToString converts an expression to a string (simplified for now)
func (e *Evaluator) expressionToString(expr parser.Expression) string {
	switch ex := expr.(type) {
	case parser.Identifier:
		// Look up in environment
		if val, ok := e.env[ex.Name]; ok {
			return val
		}
		if val, ok := os.LookupEnv(ex.Name); ok {
			return val
		}
		return ""
	case parser.StringLiteral:
		return ex.Value
	case parser.AccessId:
		switch fmt.Sprint(ex.Object) {
		case "env":
			// Look up in environment
			if val, ok := e.env[ex.Property]; ok {
				return val
			}
			if val, ok := os.LookupEnv(ex.Property); ok {
				return val
			}
			return ""
		}

		// For now, just return empty string for complex expressions
		// This will be implemented properly later
		return ""
	case parser.Or:
		// Evaluate left side first
		left := e.expressionToString(ex.Left)
		if left != "" {
			return left
		}
		// If left is empty, evaluate right
		return e.expressionToString(ex.Right)
	default:
		return ""
	}
}
