package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"miren.dev/quake/evaluator"
	"miren.dev/quake/parser"
)

func main() {
	var listTasks bool
	flag.BoolVar(&listTasks, "l", false, "List all tasks with their documentation")
	flag.Parse()

	if listTasks {
		if err := listAllTasks(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Parse arguments to support multiple tasks separated by --
	args := flag.Args()
	
	// Split arguments into groups separated by --
	var taskGroups [][]string
	currentGroup := []string{}
	
	for _, arg := range args {
		if arg == "--" {
			if len(currentGroup) > 0 {
				taskGroups = append(taskGroups, currentGroup)
				currentGroup = []string{}
			}
		} else {
			currentGroup = append(currentGroup, arg)
		}
	}
	// Add the last group if not empty
	if len(currentGroup) > 0 {
		taskGroups = append(taskGroups, currentGroup)
	}

	// If no tasks specified, run default
	if len(taskGroups) == 0 {
		if err := runTask("", nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Execute each task group in sequence
	for _, group := range taskGroups {
		taskName := group[0]
		var taskArgs []string
		if len(group) > 1 {
			taskArgs = group[1:]
		}
		
		if err := runTask(taskName, taskArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func listAllTasks() error {
	// Look for Quakefile in current directory
	quakefilePath := "Quakefile"
	if _, err := os.Stat(quakefilePath); os.IsNotExist(err) {
		return fmt.Errorf("no Quakefile found in current directory")
	}

	// Read the Quakefile
	data, err := os.ReadFile(quakefilePath)
	if err != nil {
		return fmt.Errorf("failed to read Quakefile: %w", err)
	}

	// Parse the Quakefile
	result, ok, err := parser.ParseQuakefile(string(data))
	if !ok {
		return fmt.Errorf("failed to parse Quakefile: %w", err)
	}
	if err != nil {
		return fmt.Errorf("error parsing Quakefile: %w", err)
	}

	// List all tasks
	if len(result.Tasks) == 0 {
		fmt.Println("No tasks defined in Quakefile")
		return nil
	}

	fmt.Println("Available tasks:")
	for _, task := range result.Tasks {
		// Get first line of documentation if available
		docFirstLine := getFirstLine(task.Description)

		if docFirstLine != "" {
			fmt.Printf("  %-20s %s\n", task.Name, docFirstLine)
		} else {
			fmt.Printf("  %s\n", task.Name)
		}
	}

	// Also list tasks in namespaces
	for _, namespace := range result.Namespaces {
		listNamespaceTasks(namespace, namespace.Name)
	}

	return nil
}

func listNamespaceTasks(namespace parser.Namespace, prefix string) {
	for _, task := range namespace.Tasks {
		taskName := prefix + ":" + task.Name
		docFirstLine := getFirstLine(task.Description)

		if docFirstLine != "" {
			fmt.Printf("  %-20s %s\n", taskName, docFirstLine)
		} else {
			fmt.Printf("  %s\n", taskName)
		}
	}

	// Recurse into nested namespaces
	for _, nested := range namespace.Namespaces {
		listNamespaceTasks(nested, prefix+":"+nested.Name)
	}
}

func getFirstLine(description string) string {
	if description == "" {
		return ""
	}
	// Split by newline and return first non-empty line
	lines := strings.Split(description, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func runTask(taskName string, args []string) error {
	// Look for Quakefile in current directory
	quakefilePath := "Quakefile"
	if _, err := os.Stat(quakefilePath); os.IsNotExist(err) {
		return fmt.Errorf("no Quakefile found in current directory")
	}

	// Read the Quakefile
	data, err := os.ReadFile(quakefilePath)
	if err != nil {
		return fmt.Errorf("failed to read Quakefile: %w", err)
	}

	// Parse the Quakefile
	result, ok, err := parser.ParseQuakefile(string(data))
	if !ok {
		return fmt.Errorf("failed to parse Quakefile: %w", err)
	}
	if err != nil {
		return fmt.Errorf("error parsing Quakefile: %w", err)
	}

	// Create evaluator and run task with arguments
	eval := evaluator.New(&result)
	return eval.RunTaskWithArgs(taskName, args)
}
