package evaluator

import (
	"fmt"
	"os"
	"os/exec"
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
	return &Evaluator{
		quakefile: quakefile,
		env:       make(map[string]string),
	}
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
	// Check if it's a namespaced task (contains ':')
	if strings.Contains(name, ":") {
		parts := strings.Split(name, ":")
		return e.findNamespacedTask(parts, e.quakefile.Namespaces)
	}

	// Look in top-level tasks
	for i := range e.quakefile.Tasks {
		if e.quakefile.Tasks[i].Name == name {
			return &e.quakefile.Tasks[i]
		}
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

// executeCommand runs a single command (for backward compatibility)
func (e *Evaluator) executeCommand(cmd parser.Command) error {
	return e.executeCommandWithPosition(cmd, true)
}

// executeCommandWithPosition runs a single command with position info
func (e *Evaluator) executeCommandWithPosition(cmd parser.Command, isLast bool) error {
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
