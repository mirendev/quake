package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSimpleTask(t *testing.T) {
	input := `task hello {
    echo "Hello, World!"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name: "hello",
				Commands: []Command{
					{Line: `echo "Hello, World!"`},
				},
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

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:      "greet",
				Arguments: []string{"name"},
				Commands: []Command{
					{Line: `echo "Hello, $name!"`},
				},
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

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name: "special",
				Commands: []Command{
					{Line: `echo "silent command"`, Silent: true},
					{Line: "false", ContinueOnError: true},
					{Line: `echo "normal command"`},
				},
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

	expected := QuakeFile{
		Tasks: []Task{},
	}

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