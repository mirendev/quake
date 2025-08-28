package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"miren.dev/quake/parser"
)

// [debug:parse] :: Parse a Quakefile and output its JSON representation
func ParseQuakefile(files ...string) error {
	file := "Quakefile"
	if len(files) > 0 && files[0] != "" {
		file = files[0]
	}

	// Check if file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		// If not absolute path, try to find it relative to current directory
		if !filepath.IsAbs(file) {
			// Try current directory first
			cwd, _ := os.Getwd()
			testPath := filepath.Join(cwd, file)
			if _, err := os.Stat(testPath); err == nil {
				file = testPath
			} else {
				return fmt.Errorf("file not found: %s", file)
			}
		} else {
			return fmt.Errorf("file not found: %s", file)
		}
	}

	// Read the file
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the Quakefile
	result, ok, err := parser.ParseQuakefile(string(data))
	if !ok {
		return fmt.Errorf("failed to parse Quakefile: %w", err)
	}
	if err != nil {
		return fmt.Errorf("error parsing Quakefile: %w", err)
	}

	// Convert to JSON with pretty printing
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	// Output the JSON
	fmt.Println(string(jsonData))
	return nil
}

// [debug:ast] :: Show detailed AST structure of a Quakefile
func ShowAST(files ...string) error {
	file := "Quakefile"
	if len(files) > 0 && files[0] != "" {
		file = files[0]
	}

	// Check if file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		if !filepath.IsAbs(file) {
			cwd, _ := os.Getwd()
			testPath := filepath.Join(cwd, file)
			if _, err := os.Stat(testPath); err == nil {
				file = testPath
			} else {
				return fmt.Errorf("file not found: %s", file)
			}
		} else {
			return fmt.Errorf("file not found: %s", file)
		}
	}

	// Read the file
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the Quakefile
	result, ok, err := parser.ParseQuakefile(string(data))
	if !ok {
		return fmt.Errorf("failed to parse Quakefile: %w", err)
	}
	if err != nil {
		return fmt.Errorf("error parsing Quakefile: %w", err)
	}

	fmt.Printf("=== Quakefile AST for %s ===\n\n", file)
	
	// Show tasks
	fmt.Printf("Tasks (%d):\n", len(result.Tasks))
	for _, task := range result.Tasks {
		fmt.Printf("  - %s\n", task.Name)
		if task.Description != "" {
			fmt.Printf("    Description: %s\n", task.Description)
		}
		if len(task.Arguments) > 0 {
			fmt.Printf("    Arguments: %v\n", task.Arguments)
		}
		if len(task.Dependencies) > 0 {
			fmt.Printf("    Dependencies: %v\n", task.Dependencies)
		}
		if task.IsGoTask {
			fmt.Printf("    Type: Go Task\n")
			fmt.Printf("    Source Dir: %s\n", task.GoSourceDir)
		} else {
			fmt.Printf("    Commands: %d\n", len(task.Commands))
		}
	}

	// Show namespaces
	if len(result.Namespaces) > 0 {
		fmt.Printf("\nNamespaces (%d):\n", len(result.Namespaces))
		for _, ns := range result.Namespaces {
			showNamespace(ns, "  ")
		}
	}

	// Show variables
	if len(result.Variables) > 0 {
		fmt.Printf("\nVariables (%d):\n", len(result.Variables))
		for _, v := range result.Variables {
			fmt.Printf("  - %s = %v\n", v.Name, v.Value)
		}
	}

	return nil
}

func showNamespace(ns parser.Namespace, indent string) {
	fmt.Printf("%s- %s\n", indent, ns.Name)
	if len(ns.Tasks) > 0 {
		fmt.Printf("%s  Tasks: %d\n", indent, len(ns.Tasks))
		for _, task := range ns.Tasks {
			fmt.Printf("%s    - %s\n", indent, task.Name)
		}
	}
	for _, nested := range ns.Namespaces {
		showNamespace(nested, indent+"  ")
	}
}