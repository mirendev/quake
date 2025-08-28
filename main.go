package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	flag.BoolVar(&listTasks, "l", false, "List all tasks with their documentation")
	flag.BoolVar(&verbose, "v", false, "Verbose output (show source file locations with -l)")
	flag.Parse()

	if listTasks {
		if err := listAllTasks(verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0
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

		if err := runTask(taskName, taskArgs); err != nil {
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
func findQuakefile() (string, error) {
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

func listAllTasks(verbose bool) error {
	// Look for Quakefile in current or parent directories
	quakefilePath, err := findQuakefile()
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

func runTask(taskName string, args []string) error {
	// Look for Quakefile in current or parent directories
	quakefilePath, err := findQuakefile()
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
