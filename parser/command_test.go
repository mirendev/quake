package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBacktickCommands(t *testing.T) {
	input := `task deploy {
    echo "Current commit:"
    ` + "`" + `git rev-parse --short HEAD` + "`" + `
    echo "Current date:"
    ` + "`" + `date +%Y-%m-%d` + "`" + `
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "deploy",
			Commands: []Command{
				{Line: `    echo "Current commit:"`},
				{Line: "`git rev-parse --short HEAD`", IsCommandSubstitution: true},
				{Line: `    echo "Current date:"`},
				{Line: "`date +%Y-%m-%d`", IsCommandSubstitution: true},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseBacktickWithPrefixes(t *testing.T) {
	input := `task info {
    @` + "`" + `echo "Silent command"` + "`" + `
    -` + "`" + `false || true` + "`" + `
    ` + "`" + `pwd` + "`" + `
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "info",
			Commands: []Command{
				{Line: "`echo \"Silent command\"`", Silent: true, IsCommandSubstitution: true},
				{Line: "`false || true`", ContinueOnError: true, IsCommandSubstitution: true},
				{Line: "`pwd`", IsCommandSubstitution: true},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseMixedCommandsAndBackticks(t *testing.T) {
	input := `task build {
    echo "Building..."
    ` + "`" + `make clean` + "`" + `
    make build
    ` + "`" + `make test` + "`" + `
    echo "Done"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "build",
			Commands: []Command{
				{Line: `    echo "Building..."`},
				{Line: "`make clean`", IsCommandSubstitution: true},
				{Line: `    make build`},
				{Line: "`make test`", IsCommandSubstitution: true},
				{Line: `    echo "Done"`},
			},
		},
	}

	require.Equal(t, expected, result)
}
