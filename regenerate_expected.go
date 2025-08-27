package main

import (
	"encoding/json"
	"fmt"
	"miren.dev/quake/parser"
	"os"
)

func main() {
	// Regenerate complex expected
	data, err := os.ReadFile("/home/evanphx/mn-git/quake/testdata/complex/Quakefile")
	if err != nil {
		fmt.Printf("Could not read complex file: %v\n", err)
		return
	}

	result, ok, err := parser.ParseQuakefile(string(data))
	if !ok {
		fmt.Printf("Parse failed: %v\n", err)
		return
	}
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Check if any tasks have descriptions
	tasksWithDesc := 0
	for _, task := range result.Tasks {
		if task.Description != "" {
			tasksWithDesc++
			fmt.Printf("Task '%s' has description: %s\n", task.Name, task.Description)
		}
	}
	fmt.Printf("Found %d tasks with descriptions\n", tasksWithDesc)

	// Generate expected JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("JSON marshal error: %v\n", err)
		return
	}

	// Write to expected output file
	err = os.WriteFile("/home/evanphx/mn-git/quake/testdata/complex/expected_ast.json", jsonData, 0644)
	if err != nil {
		fmt.Printf("Write error: %v\n", err)
		return
	}

	fmt.Println("Updated testdata/complex/expected_ast.json")
}
