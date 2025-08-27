package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// Helper function to create a QuakeFile with initialized slices
func makeQuakeFile() QuakeFile {
	return QuakeFile{
		Tasks:      []Task{},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}
}

func TestParseSimpleTask(t *testing.T) {
	input := `task hello {
    echo "Hello, World!"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "hello",
			Commands: []Command{
				{Line: `    echo "Hello, World!"`},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseTaskWithArguments(t *testing.T) {
	input := `task greet(name) {
    echo "Hello, $name!"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name:      "greet",
			Arguments: []string{"name"},
			Commands: []Command{
				{Line: `    echo "Hello, $name!"`},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseTaskWithSpecialCommands(t *testing.T) {
	input := `task special {
    @echo "silent command"
    -false
    echo "normal command"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "special",
			Commands: []Command{
				{Line: `    echo "silent command"`, Silent: true},
				{Line: `    false`, ContinueOnError: true},
				{Line: `    echo "normal command"`},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseEmptyFile(t *testing.T) {
	input := ""

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()

	require.Equal(t, expected, result)
}

func TestJSONSerialization(t *testing.T) {
	input := `task hello {
    echo "world"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	// Test that we can serialize to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	require.NoError(t, err, "should serialize to JSON")

	// Test that we can deserialize back
	var deserialized QuakeFile
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err, "should deserialize from JSON")
	require.Equal(t, result, deserialized, "should round-trip through JSON")
}

func TestParseSimpleNamespace(t *testing.T) {
	input := `namespace db {
    task migrate {
        echo "Running migrations"
    }
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Namespaces = []Namespace{
		{
			Name: "db",
			Tasks: []Task{
				{
					Name: "migrate",
					Commands: []Command{
						{Line: `        echo "Running migrations"`},
					},
				},
			},
			Namespaces: []Namespace{},
			Variables:  []Variable{},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseFileNamespace(t *testing.T) {
	input := `namespace api

task start {
    echo "Starting API server"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.FileNamespace = "api"
	expected.Tasks = []Task{
		{
			Name: "start",
			Commands: []Command{
				{Line: `    echo "Starting API server"`},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseTaskWithDependencies(t *testing.T) {
	input := `task deploy => build, test {
    echo "Deploying..."
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name:         "deploy",
			Dependencies: []string{"build", "test"},
			Commands: []Command{
				{Line: `    echo "Deploying..."`},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseTaskWithQuotedBraces(t *testing.T) {
	input := `task test {
    echo "This has } inside quotes"
    echo 'Single quotes with } too'
    echo "Multiple } braces } in one line"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "test",
			Commands: []Command{
				{Line: `    echo "This has } inside quotes"`},
				{Line: `    echo 'Single quotes with } too'`},
				{Line: `    echo "Multiple } braces } in one line"`},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseTaskWithNestedBraces(t *testing.T) {
	input := `task complex {
    if [ -f file.txt ]; then
        echo "File exists { with braces }"
    fi
    echo "Done"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "complex",
			Commands: []Command{
				{Line: `    if [ -f file.txt ]; then`},
				{Line: `        echo "File exists { with braces }"`},
				{Line: `    fi`},
				{Line: `    echo "Done"`},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseTaskWithJSONInCommand(t *testing.T) {
	input := `task json {
    curl -d '{"key": "value", "nested": {"inner": "data"}}' api.com
    echo "JSON sent"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "json",
			Commands: []Command{
				{Line: `    curl -d '{"key": "value", "nested": {"inner": "data"}}' api.com`},
				{Line: `    echo "JSON sent"`},
			},
		},
	}

	require.Equal(t, expected, result)
}
