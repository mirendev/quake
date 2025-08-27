package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSimpleVariables(t *testing.T) {
	input := `VERSION = "1.2.3"
APP_NAME = "myapp"
BUILD_DIR = "build"

task info {
    echo "App: $APP_NAME v$VERSION"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Variables: []Variable{
			{Name: "VERSION", Value: `"1.2.3"`},
			{Name: "APP_NAME", Value: `"myapp"`},
			{Name: "BUILD_DIR", Value: `"build"`},
		},
		Tasks: []Task{
			{
				Name: "info",
				Commands: []Command{
					{Line: `    echo "App: $APP_NAME v$VERSION"`},
				},
			},
		},
		Namespaces: []Namespace{},
	}

	require.Equal(t, expected, result)
}

func TestParseCommandSubstitution(t *testing.T) {
	input := `GIT_COMMIT = ` + "`" + `git rev-parse --short HEAD` + "`" + `
BUILD_DATE = ` + "`" + `date +%Y-%m-%d` + "`" + `

task version {
    echo "Commit: $GIT_COMMIT"
    echo "Date: $BUILD_DATE"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Variables: []Variable{
			{Name: "GIT_COMMIT", Value: "`git rev-parse --short HEAD`", CommandSubstitution: true},
			{Name: "BUILD_DATE", Value: "`date +%Y-%m-%d`", CommandSubstitution: true},
		},
		Tasks: []Task{
			{
				Name: "version",
				Commands: []Command{
					{Line: `    echo "Commit: $GIT_COMMIT"`},
					{Line: `    echo "Date: $BUILD_DATE"`},
				},
			},
		},
		Namespaces: []Namespace{},
	}

	require.Equal(t, expected, result)
}

func TestParseExpressionVariables(t *testing.T) {
	input := `DEPLOY_ENV = {{env.DEPLOY_ENV || "development"}}
API_KEY = {{env.API_KEY || "default-key"}}

task deploy {
    echo "Env: $DEPLOY_ENV"
    echo "Key: $API_KEY"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Variables: []Variable{
			{Name: "DEPLOY_ENV", Value: `{{env.DEPLOY_ENV || "development"}}`, IsExpression: true},
			{Name: "API_KEY", Value: `{{env.API_KEY || "default-key"}}`, IsExpression: true},
		},
		Tasks: []Task{
			{
				Name: "deploy",
				Commands: []Command{
					{Line: `    echo "Env: $DEPLOY_ENV"`},
					{Line: `    echo "Key: $API_KEY"`},
				},
			},
		},
		Namespaces: []Namespace{},
	}

	require.Equal(t, expected, result)
}

func TestParseMultilineStringVariable(t *testing.T) {
	input := `HELP_TEXT = """
Usage: quake [task]

Tasks:
  build  Build the app
  test   Run tests
"""

task help {
    echo "$HELP_TEXT"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Variables: []Variable{
			{
				Name:        "HELP_TEXT",
				Value:       "Usage: quake [task]\n\nTasks:\n  build  Build the app\n  test   Run tests\n",
				IsMultiline: true,
			},
		},
		Tasks: []Task{
			{
				Name: "help",
				Commands: []Command{
					{Line: `    echo "$HELP_TEXT"`},
				},
			},
		},
		Namespaces: []Namespace{},
	}

	require.Equal(t, expected, result)
}

func TestParseTaskLocalVariables(t *testing.T) {
	input := `task build(target) {
    TARGET = {{target || "release"}}
    echo "Building $TARGET"
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:      "build",
				Arguments: []string{"target"},
				Commands: []Command{
					{Line: `    TARGET = {{target || "release"}}`},
					{Line: `    echo "Building $TARGET"`},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}

func TestParseNamespaceVariables(t *testing.T) {
	input := `namespace docker {
    IMAGE_NAME = "myapp"
    IMAGE_TAG = "latest"
    
    task build {
        echo "Building $IMAGE_NAME:$IMAGE_TAG"
    }
}`

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Namespaces: []Namespace{
			{
				Name: "docker",
				Variables: []Variable{
					{Name: "IMAGE_NAME", Value: `"myapp"`},
					{Name: "IMAGE_TAG", Value: `"latest"`},
				},
				Tasks: []Task{
					{
						Name: "build",
						Commands: []Command{
							{Line: `        echo "Building $IMAGE_NAME:$IMAGE_TAG"`},
						},
					},
				},
				Namespaces: []Namespace{},
			},
		},
		Tasks:     []Task{},
		Variables: []Variable{},
	}

	require.Equal(t, expected, result)
}
