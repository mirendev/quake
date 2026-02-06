package evaluator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"miren.dev/quake/parser"
)

// TestBug_EnvAccessIdResolvesToEmpty demonstrates that {{env.VAR}} expressions
// silently resolve to empty string because expressionToString uses
// fmt.Sprint(ex.Object) on an Identifier struct, which produces "{env}"
// instead of "env", so the switch case never matches.
func TestBug_EnvAccessIdResolvesToEmpty(t *testing.T) {
	os.Setenv("QUAKE_TEST_VAR", "hello_from_env")
	defer os.Unsetenv("QUAKE_TEST_VAR")

	qf := &parser.QuakeFile{
		Tasks:      []parser.Task{},
		Namespaces: []parser.Namespace{},
		Variables:  []parser.Variable{},
	}
	ev := New(qf)

	// Build the expression that the parser produces for env.QUAKE_TEST_VAR
	expr := parser.AccessId{
		Object:   parser.Identifier{Name: "env"},
		Property: "QUAKE_TEST_VAR",
	}

	result := ev.expressionToString(expr)
	require.Equal(t, "hello_from_env", result)
}
