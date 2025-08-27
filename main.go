package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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

	// For now, just show usage if no flags provided
	fmt.Println("Usage: quake [options]")
	fmt.Println("Options:")
	fmt.Println("  -l    List all tasks with their documentation")
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
