package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"miren.dev/mflags"
	"miren.dev/quake/evaluator"
	"miren.dev/quake/internal/gotasks"
	"miren.dev/quake/parser"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	// Ensure cleanup on exit
	defer func() {
		if globalTaskCache != nil {
			globalTaskCache.Cleanup()
		}
	}()

	var listTasks bool
	var verbose bool
	var generateTask bool
	var initQuakefile bool
	var quakefilePath string

	flags := mflags.NewFlagSet("quake")
	flags.BoolVar(&listTasks, "list", 'l', false, "List all tasks with their documentation")
	flags.BoolVar(&verbose, "", 'v', false, "Verbose output (show source file locations with -l)")
	flags.BoolVar(&generateTask, "generate", 'g', false, "Generate a new task using Claude AI")
	flags.BoolVar(&initQuakefile, "init", 0, false, "Initialize a new Quakefile using Claude AI")
	flags.StringVar(&quakefilePath, "file", 'f', "", "Path to Quakefile (default: search for Quakefile in current and parent directories)")

	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, mflags.ErrHelp) {
			return 1
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if initQuakefile {
		if err := initQuakefileWithClaude(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	}

	if generateTask {
		if err := generateTaskWithClaude(quakefilePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	}

	if listTasks {
		if err := listAllTasks(verbose, quakefilePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	}

	// Parse arguments to support multiple tasks separated by --
	args := flags.Args()

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
		if err := runTask("", nil, quakefilePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	}

	// Execute each task group in sequence
	for _, group := range taskGroups {
		taskName := group[0]
		var taskArgs []string
		if len(group) > 1 {
			taskArgs = group[1:]
		}

		if err := runTask(taskName, taskArgs, quakefilePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}

	return 0
}

// findQuakeFiles finds all .quake files in the qtasks directories
func findQuakeFiles(baseDir string) []string {
	var quakeFiles []string

	// Directories to search for .quake files
	taskDirs := []string{
		filepath.Join(baseDir, "qtasks"),
		filepath.Join(baseDir, "lib", "qtasks"),
		filepath.Join(baseDir, "internal", "qtasks"),
	}

	for _, dir := range taskDirs {
		// Check if directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		// Find all .quake files in the directory
		files, err := filepath.Glob(filepath.Join(dir, "*.quake"))
		if err != nil {
			continue
		}

		quakeFiles = append(quakeFiles, files...)
	}

	return quakeFiles
}

// mergeQuakefiles merges multiple QuakeFile structs into one
func mergeQuakefiles(files ...parser.QuakeFile) parser.QuakeFile {
	result := parser.QuakeFile{}

	for _, file := range files {
		result.Tasks = append(result.Tasks, file.Tasks...)
		result.Variables = append(result.Variables, file.Variables...)
		result.Namespaces = append(result.Namespaces, file.Namespaces...)
	}

	return result
}

// Global task cache that will be cleaned up on exit
var globalTaskCache *gotasks.TaskCache

// discoverGoTasks finds and prepares Go tasks in all qtasks directories
func discoverGoTasks(baseDir string) ([]parser.Task, error) {
	var allTasks []parser.Task

	// Directories to search for Go tasks (same as .quake files)
	taskDirs := []string{
		filepath.Join(baseDir, "qtasks"),
		filepath.Join(baseDir, "lib", "qtasks"),
		filepath.Join(baseDir, "internal", "qtasks"),
	}

	// Create task cache if not exists
	if globalTaskCache == nil {
		var err error
		globalTaskCache, err = gotasks.NewTaskCache()
		if err != nil {
			return nil, fmt.Errorf("failed to create task cache: %w", err)
		}
	}

	for _, qtasksDir := range taskDirs {
		// Check if directory exists
		if _, err := os.Stat(qtasksDir); os.IsNotExist(err) {
			continue
		}

		// Discover Go functions in this directory
		taskFuncs, err := gotasks.DiscoverTasks(qtasksDir)
		if err != nil {
			// Warning but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to discover Go tasks in %s: %v\n", qtasksDir, err)
			continue
		}

		if len(taskFuncs) == 0 {
			// No Go tasks in this directory
			continue
		}

		// Get the dispatcher path for this directory's tasks
		dispatcherPath, err := globalTaskCache.GetDispatcherPath(taskFuncs, qtasksDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to generate dispatcher for %s: %v\n", qtasksDir, err)
			continue
		}

		// Convert discovered functions to Task structs for this directory
		for _, fn := range taskFuncs {
			// Use extracted comment as description, or fall back to generic description
			description := fn.Description
			if description == "" {
				description = fmt.Sprintf("Go task from %s", filepath.Base(fn.SourceFile))
			}

			task := parser.Task{
				Name:         fn.Name,
				Description:  description,
				Arguments:    fn.Params,
				IsGoTask:     true,
				GoDispatcher: dispatcherPath,
				GoSourceDir:  qtasksDir,
				SourceFile:   fn.SourceFile,
				Commands:     []parser.Command{}, // Go tasks don't have shell commands
			}

			// If task has a namespace, prepend it to the name
			if fn.Namespace != "" {
				task.Name = fn.Namespace + ":" + task.Name
			}

			allTasks = append(allTasks, task)
		}
	}

	return allTasks, nil
}

// loadAllQuakefiles loads and merges the main Quakefile with all .quake files
func loadAllQuakefiles(mainPath string) (parser.QuakeFile, error) {
	// Read and parse the main Quakefile
	data, err := os.ReadFile(mainPath)
	if err != nil {
		return parser.QuakeFile{}, fmt.Errorf("failed to read Quakefile: %w", err)
	}

	mainResult, ok, err := parser.ParseQuakefileWithSource(string(data), mainPath)
	if !ok {
		return parser.QuakeFile{}, fmt.Errorf("failed to parse Quakefile: %w", err)
	}
	if err != nil {
		return parser.QuakeFile{}, fmt.Errorf("error parsing Quakefile: %w", err)
	}

	// Find and load .quake files from qtasks directories
	baseDir := filepath.Dir(mainPath)
	quakeFiles := findQuakeFiles(baseDir)

	var additionalResults []parser.QuakeFile
	for _, qfile := range quakeFiles {
		data, err := os.ReadFile(qfile)
		if err != nil {
			// Skip files that can't be read
			fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", qfile, err)
			continue
		}

		result, ok, err := parser.ParseQuakefileWithSource(string(data), qfile)
		if !ok || err != nil {
			// Skip files that can't be parsed
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", qfile, err)
			continue
		}

		additionalResults = append(additionalResults, result)
	}

	// Discover and add Go tasks
	goTasks, err := discoverGoTasks(baseDir)
	if err != nil {
		// Warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: failed to discover Go tasks: %v\n", err)
	} else if len(goTasks) > 0 {
		// Add Go tasks as a separate QuakeFile
		goTasksFile := parser.QuakeFile{
			Tasks: goTasks,
		}
		additionalResults = append(additionalResults, goTasksFile)
	}

	// Merge all results
	allResults := append([]parser.QuakeFile{mainResult}, additionalResults...)
	return mergeQuakefiles(allResults...), nil
}

// findQuakefile searches for a Quakefile in the current directory and parent directories
// If customPath is provided, it validates and returns that path instead
func findQuakefile(customPath string) (string, error) {
	// If a custom path was provided, use it
	if customPath != "" {
		// Convert to absolute path if relative
		absPath, err := filepath.Abs(customPath)
		if err != nil {
			return "", fmt.Errorf("invalid path %s: %w", customPath, err)
		}

		// Check if file exists
		if _, err := os.Stat(absPath); err != nil {
			return "", fmt.Errorf("Quakefile not found at %s: %w", absPath, err)
		}

		return absPath, nil
	}

	// Default behavior: search current and parent directories
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		quakefilePath := filepath.Join(dir, "Quakefile")
		if _, err := os.Stat(quakefilePath); err == nil {
			return quakefilePath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root directory
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no Quakefile found in current directory or any parent directory")
}

func listAllTasks(verbose bool, customPath string) error {
	// Look for Quakefile in current or parent directories
	quakefilePath, err := findQuakefile(customPath)
	if err != nil {
		return err
	}

	// Load all quakefiles (main + qtasks directories)
	result, err := loadAllQuakefiles(quakefilePath)
	if err != nil {
		return err
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

		if verbose && task.SourceFile != "" {
			// Show source file in verbose mode (relative to current directory)
			cwd, _ := os.Getwd()
			relPath, err := filepath.Rel(cwd, task.SourceFile)
			if err != nil {
				relPath = task.SourceFile // fallback to absolute path
			}
			if docFirstLine != "" {
				fmt.Printf("  %-20s %s [%s]\n", task.Name, docFirstLine, relPath)
			} else {
				fmt.Printf("  %-20s [%s]\n", task.Name, relPath)
			}
		} else {
			// Normal mode
			if docFirstLine != "" {
				fmt.Printf("  %-20s %s\n", task.Name, docFirstLine)
			} else {
				fmt.Printf("  %s\n", task.Name)
			}
		}
	}

	// Also list tasks in namespaces
	for _, namespace := range result.Namespaces {
		listNamespaceTasks(namespace, namespace.Name, verbose)
	}

	return nil
}

func listNamespaceTasks(namespace parser.Namespace, prefix string, verbose bool) {
	for _, task := range namespace.Tasks {
		taskName := prefix + ":" + task.Name
		docFirstLine := getFirstLine(task.Description)

		if verbose && task.SourceFile != "" {
			// Show source file in verbose mode (relative to current directory)
			cwd, _ := os.Getwd()
			relPath, err := filepath.Rel(cwd, task.SourceFile)
			if err != nil {
				relPath = task.SourceFile // fallback to absolute path
			}
			if docFirstLine != "" {
				fmt.Printf("  %-20s %s [%s]\n", taskName, docFirstLine, relPath)
			} else {
				fmt.Printf("  %-20s [%s]\n", taskName, relPath)
			}
		} else {
			// Normal mode
			if docFirstLine != "" {
				fmt.Printf("  %-20s %s\n", taskName, docFirstLine)
			} else {
				fmt.Printf("  %s\n", taskName)
			}
		}
	}

	// Recurse into nested namespaces
	for _, nested := range namespace.Namespaces {
		listNamespaceTasks(nested, prefix+":"+nested.Name, verbose)
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

func runTask(taskName string, args []string, customPath string) error {
	// Look for Quakefile in current or parent directories
	quakefilePath, err := findQuakefile(customPath)
	if err != nil {
		return err
	}

	// Change to the directory containing the Quakefile
	quakefileDir := filepath.Dir(quakefilePath)
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if quakefileDir != originalDir {
		if err := os.Chdir(quakefileDir); err != nil {
			return fmt.Errorf("failed to change to Quakefile directory: %w", err)
		}
		// Change back to original directory when done
		defer os.Chdir(originalDir)
	}

	// Load all quakefiles (main + qtasks directories)
	result, err := loadAllQuakefiles(quakefilePath)
	if err != nil {
		return err
	}

	// Create evaluator and run task with arguments
	eval := evaluator.New(&result)
	return eval.RunTaskWithArgs(taskName, args)
}

// extractTaskFromOutput extracts a task definition from Claude's output
// It handles both plain output and markdown code blocks
func extractTaskFromOutput(output string) string {
	output = strings.TrimSpace(output)

	// First, check if the output is wrapped in code blocks
	// Pattern for ```quake or ``` blocks
	codeBlockRe := regexp.MustCompile("(?s)```(?:quake.*)?\\s*\n(.*?)```")
	matches := codeBlockRe.FindStringSubmatch(output)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// If no code blocks, check if it starts with "task" (valid task definition)
	if strings.HasPrefix(output, "task ") || strings.HasPrefix(output, "#") {
		// It looks like a raw task definition
		return output
	}

	// Try to find a task definition anywhere in the output
	// Look for lines starting with "task "
	lines := strings.Split(output, "\n")
	var taskLines []string
	inTask := false
	braceCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start capturing when we see "task "
		if !inTask && strings.HasPrefix(trimmed, "task ") {
			inTask = true
			taskLines = append(taskLines, line)
			// Count braces in the first line
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			continue
		}

		// If we're in a task, keep capturing
		if inTask {
			taskLines = append(taskLines, line)
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Stop when braces are balanced
			if braceCount == 0 {
				break
			}
		}
	}

	if len(taskLines) > 0 {
		return strings.Join(taskLines, "\n")
	}

	// If nothing worked, return the original output and let the user see it
	return output
}

// generateTaskWithClaude prompts the user for a task description and uses Claude to generate it
func generateTaskWithClaude(customPath string) error {
	// Check if claude CLI is available
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		// Try common locations
		possiblePaths := []string{
			"/usr/local/bin/claude",
			"/usr/bin/claude",
			filepath.Join(os.Getenv("HOME"), "bin", "claude"),
			filepath.Join(os.Getenv("HOME"), ".local", "bin", "claude"),
		}

		found := false
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				claudePath = path
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("claude CLI not found. Please ensure 'claude' is installed and in your PATH")
		}
	}

	// Prompt user for task description
	fmt.Print("Describe the task you want to create: ")
	reader := bufio.NewReader(os.Stdin)
	taskDescription, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read task description: %w", err)
	}
	taskDescription = strings.TrimSpace(taskDescription)

	if taskDescription == "" {
		return fmt.Errorf("task description cannot be empty")
	}

	// Find the Quakefile
	quakefilePath, err := findQuakefile(customPath)
	if err != nil {
		return err
	}

	// Read the current Quakefile
	currentContent, err := os.ReadFile(quakefilePath)
	if err != nil {
		return fmt.Errorf("failed to read Quakefile: %w", err)
	}

	// Create the prompt for Claude
	prompt := fmt.Sprintf(`You are a helpful assistant that creates tasks for Quakefile build systems.

QUAKEFILE SYNTAX RULES:
1. Tasks are defined with: task <name> { ... }
2. Tasks can have dependencies: task build => test { ... }
3. Tasks can have arguments: task deploy(environment) { ... }
4. Tasks can have both: task deploy(env) => build, test { ... }
5. Commands in tasks are shell commands, one per line
6. Comments start with #
7. Variables can be referenced with $VAR or {{expression}}
8. Command substitution uses backticks: `+"`command`"+`
9. Silent commands start with @
10. Continue on error with -

The user wants to add this task: "%s"

Current Quakefile content:
%s

Please generate ONLY the new task definition to add to this Quakefile.

Requirements:
- Output ONLY the task code, no explanations
- Use descriptive comments
- Follow the existing style and conventions
- Make the task name appropriate and consistent with existing tasks
- If the task seems like it should have dependencies on existing tasks, include them`,
		taskDescription, string(currentContent))

	// Execute claude with the prompt
	cmd := exec.Command(claudePath, "-p")
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stderr = os.Stderr

	var out bytes.Buffer
	cmd.Stdout = &out

	fmt.Println("Generating task with Claude...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run claude: %w", err)
	}

	// Extract the task from the output
	generatedTask := extractTaskFromOutput(out.String())
	if generatedTask == "" {
		return fmt.Errorf("claude returned empty response or no valid task found")
	}

	// Show the generated task to the user
	fmt.Println("\nGenerated task:")
	fmt.Println("---")
	fmt.Println(generatedTask)
	fmt.Println("---")

	// Ask for confirmation
	fmt.Print("\nAdd this task to the Quakefile? (y/n): ")
	confirmation, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	confirmation = strings.ToLower(strings.TrimSpace(confirmation))

	if confirmation != "y" && confirmation != "yes" {
		fmt.Println("Task not added.")
		return nil
	}

	// Append the task to the Quakefile
	updatedContent := string(currentContent)
	if !strings.HasSuffix(updatedContent, "\n") {
		updatedContent += "\n"
	}
	updatedContent += "\n" + generatedTask + "\n"

	// Write the updated Quakefile
	if err := os.WriteFile(quakefilePath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated Quakefile: %w", err)
	}

	fmt.Printf("✅ Task added to %s\n", quakefilePath)
	return nil
}

// analyzeProjectContext examines the current directory to gather context about the project
func analyzeProjectContext() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	var analysis strings.Builder
	analysis.WriteString("PROJECT ANALYSIS:\n\n")

	// Detect build system and configuration files
	buildFiles := []string{
		"go.mod",             // Go
		"package.json",       // Node.js
		"Cargo.toml",         // Rust
		"pom.xml",            // Maven (Java)
		"build.gradle",       // Gradle (Java/Kotlin)
		"Makefile",           // Make
		"CMakeLists.txt",     // CMake (C/C++)
		"setup.py",           // Python
		"pyproject.toml",     // Python
		"Gemfile",            // Ruby
		"composer.json",      // PHP
		"build.sbt",          // Scala
		"mix.exs",            // Elixir
		"Dockerfile",         // Docker
		"docker-compose.yml", // Docker Compose
	}

	var detectedFiles []string
	for _, file := range buildFiles {
		if _, err := os.Stat(filepath.Join(cwd, file)); err == nil {
			detectedFiles = append(detectedFiles, file)
		}
	}

	if len(detectedFiles) > 0 {
		analysis.WriteString("Detected build/config files:\n")
		for _, file := range detectedFiles {
			analysis.WriteString(fmt.Sprintf("  - %s\n", file))
		}
		analysis.WriteString("\n")
	}

	// Detect programming languages by file extensions
	languageFiles := make(map[string]int)
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories and common ignore patterns
		if info.IsDir() {
			name := filepath.Base(path)
			if strings.HasPrefix(name, ".") ||
				name == "node_modules" ||
				name == "vendor" ||
				name == "target" ||
				name == "build" ||
				name == "dist" {
				return filepath.SkipDir
			}
			// Only go 3 levels deep
			relPath, _ := filepath.Rel(cwd, path)
			if strings.Count(relPath, string(os.PathSeparator)) > 3 {
				return filepath.SkipDir
			}
			return nil
		}

		// Count files by extension
		ext := filepath.Ext(path)
		if ext != "" {
			languageFiles[ext]++
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to analyze project structure: %w", err)
	}

	// Map extensions to languages
	extensionToLanguage := map[string]string{
		".go":    "Go",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".py":    "Python",
		".rb":    "Ruby",
		".rs":    "Rust",
		".java":  "Java",
		".kt":    "Kotlin",
		".c":     "C",
		".cpp":   "C++",
		".h":     "C/C++ headers",
		".cs":    "C#",
		".php":   "PHP",
		".swift": "Swift",
		".m":     "Objective-C",
		".scala": "Scala",
		".ex":    "Elixir",
		".exs":   "Elixir",
	}

	if len(languageFiles) > 0 {
		analysis.WriteString("Detected programming languages (by file count):\n")
		// Sort by count
		type langCount struct {
			lang  string
			count int
		}
		var langs []langCount
		for ext, count := range languageFiles {
			if lang, ok := extensionToLanguage[ext]; ok && count > 0 {
				langs = append(langs, langCount{lang, count})
			}
		}
		// Simple sort by count (descending)
		for i := 0; i < len(langs); i++ {
			for j := i + 1; j < len(langs); j++ {
				if langs[j].count > langs[i].count {
					langs[i], langs[j] = langs[j], langs[i]
				}
			}
		}
		for _, lc := range langs {
			analysis.WriteString(fmt.Sprintf("  - %s (%d files)\n", lc.lang, lc.count))
		}
		analysis.WriteString("\n")
	}

	// List top-level directory structure
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var dirs []string
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files and common directories
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, name+"/")
		} else {
			files = append(files, name)
		}
	}

	if len(dirs) > 0 || len(files) > 0 {
		analysis.WriteString("Top-level directory structure:\n")
		for _, dir := range dirs {
			analysis.WriteString(fmt.Sprintf("  %s\n", dir))
		}
		for _, file := range files {
			analysis.WriteString(fmt.Sprintf("  %s\n", file))
		}
	}

	return analysis.String(), nil
}

// initQuakefileWithClaude analyzes the project and uses Claude to generate an initial Quakefile
func initQuakefileWithClaude() error {
	// Check if a Quakefile already exists
	existingPath, err := findQuakefile("")
	if err == nil {
		// A Quakefile was found
		cwd, _ := os.Getwd()
		relPath, _ := filepath.Rel(cwd, existingPath)
		if relPath == "" {
			relPath = existingPath
		}
		return fmt.Errorf("a Quakefile already exists at %s\nRemove it first or use 'quake -g' to add tasks to it", relPath)
	}

	// Check if claude CLI is available
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		// Try common locations
		possiblePaths := []string{
			"/usr/local/bin/claude",
			"/usr/bin/claude",
			filepath.Join(os.Getenv("HOME"), "bin", "claude"),
			filepath.Join(os.Getenv("HOME"), ".local", "bin", "claude"),
		}

		found := false
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				claudePath = path
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("claude CLI not found. Please ensure 'claude' is installed and in your PATH")
		}
	}

	fmt.Println("Analyzing project structure...")

	// Analyze the project
	projectContext, err := analyzeProjectContext()
	if err != nil {
		return fmt.Errorf("failed to analyze project: %w", err)
	}

	// Create the prompt for Claude
	prompt := fmt.Sprintf(`You are a helpful assistant that creates Quakefile build system configurations.

QUAKEFILE SYNTAX RULES:
1. Tasks are defined with: task <name> { ... }
2. Tasks can have dependencies: task build => test { ... }
3. Tasks can have arguments: task deploy(environment) { ... }
4. Tasks can have both: task deploy(env) => build, test { ... }
5. Commands in tasks are shell commands, one per line
6. Comments start with #
7. Silent commands start with @
8. Continue on error with -
9. Tasks can be organized in namespaces: namespace docker { task build { ... } }

VARIABLE USAGE (IMPORTANT):
Variables in Quakefile work differently than shell variables!

1. DEFINING variables (at top level, outside tasks):
   - String literals: VERSION = "1.0.0"
   - Command substitution: GIT_COMMIT = `+"`git rev-parse HEAD`"+`
   - Expressions: BUILD_TIME = `+"`date -u +\"%Y-%m-%dT%H:%M:%SZ\"`"+`

2. REFERENCING variables in shell commands (inside tasks):
   - Use $VAR for Quakefile variables: echo "Version: $VERSION"
   - Use ${VAR} for environment variables: echo "User: ${USER}"
   - Use {{expression}} for complex expressions: NAME = {{name || "default"}}
   - Use {{env.VAR}} for environment variables: DB_NAME = {{env.DB_NAME || "myapp_dev"}}

3. EXAMPLES:
   Good:
     VERSION = "1.0.0"
     task version {
         echo "Version: $VERSION"
     }

   Good:
     PROJECT = "myapp"
     BUILD_DIR = "build"
     task build {
         mkdir -p $BUILD_DIR
         go build -o $BUILD_DIR/$PROJECT
     }

   Good (with command substitution):
     GIT_COMMIT = `+"`git rev-parse HEAD`"+`
     task info {
         echo "Commit: $GIT_COMMIT"
     }

   Bad (don't mix shell variable syntax):
     VERSION="1.0.0"  # Wrong - this is shell syntax, not Quakefile
     task build {
         VERSION="1.0.0"  # Wrong - define variables at top level
         echo $VERSION
     }

COMMON TASK PATTERNS:
- Default task: task default { ... } or task default => build
- Build/compile tasks with dependencies on lint/test
- Clean tasks to remove build artifacts
- Test tasks with coverage options
- Lint/format tasks for code quality
- Run/watch tasks for development
- Deploy tasks with environment arguments
- Docker tasks in docker namespace
- Database tasks in db namespace

%s

Please generate a comprehensive initial Quakefile for this project.

Requirements:
- Output ONLY the Quakefile content, no explanations or markdown
- Create appropriate tasks based on the detected project type
- Include a helpful default task
- Add descriptive comments for each task
- Use appropriate dependencies between tasks
- Include common development workflows (build, test, run, clean, etc.)
- Follow best practices for the detected languages and tools
- Use namespaces for logical grouping when appropriate
- Make it production-ready and useful from day one`, projectContext)

	// Execute claude with the prompt
	cmd := exec.Command(claudePath, "-p")
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stderr = os.Stderr

	var out bytes.Buffer
	cmd.Stdout = &out

	fmt.Println("Generating Quakefile with Claude...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run claude: %w", err)
	}

	// Extract the Quakefile from the output
	generatedQuakefile := extractTaskFromOutput(out.String())
	if generatedQuakefile == "" {
		return fmt.Errorf("claude returned empty response or no valid Quakefile found")
	}

	// Show the generated Quakefile to the user
	fmt.Println("\nGenerated Quakefile:")
	fmt.Println("---")
	fmt.Println(generatedQuakefile)
	fmt.Println("---")

	// Ask for confirmation
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nCreate this Quakefile? (y/n): ")
	confirmation, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	confirmation = strings.ToLower(strings.TrimSpace(confirmation))

	if confirmation != "y" && confirmation != "yes" {
		fmt.Println("Quakefile not created.")
		return nil
	}

	// Write the Quakefile to the current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	quakefilePath := filepath.Join(cwd, "Quakefile")
	if err := os.WriteFile(quakefilePath, []byte(generatedQuakefile), 0644); err != nil {
		return fmt.Errorf("failed to write Quakefile: %w", err)
	}

	fmt.Printf("\n✅ Quakefile created at %s\n", quakefilePath)
	fmt.Println("\nNext steps:")
	fmt.Println("  quake -l          # List available tasks")
	fmt.Println("  quake <task>      # Run a specific task")
	fmt.Println("  quake             # Run the default task")
	return nil
}
