package parser

import "encoding/json"

// QuakeFile represents the root of a parsed Quakefile
type QuakeFile struct {
	Tasks         []Task      `json:"tasks"`
	Namespaces    []Namespace `json:"namespaces,omitempty"`
	Variables     []Variable  `json:"variables,omitempty"`
	FileNamespace string      `json:"file_namespace,omitempty"`
}

// UnmarshalJSON ensures empty slices are initialized correctly
func (q *QuakeFile) UnmarshalJSON(data []byte) error {
	type Alias QuakeFile
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(q),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	// Initialize nil slices to empty slices
	if q.Tasks == nil {
		q.Tasks = []Task{}
	}
	if q.Namespaces == nil {
		q.Namespaces = []Namespace{}
	}
	if q.Variables == nil {
		q.Variables = []Variable{}
	}
	return nil
}

// Task represents a task definition in a Quakefile
type Task struct {
	Name         string    `json:"name"`
	Arguments    []string  `json:"arguments,omitempty"`
	Dependencies []string  `json:"dependencies,omitempty"`
	Commands     []Command `json:"commands"`
}

// Variable represents a variable assignment
type Variable struct {
	Name                string `json:"name"`
	Value               string `json:"value"`
	IsExpression        bool   `json:"is_expression,omitempty"`
	CommandSubstitution bool   `json:"command_substitution,omitempty"`
	IsMultiline         bool   `json:"is_multiline,omitempty"`
}

// Namespace represents a namespace block containing tasks and nested namespaces
type Namespace struct {
	Name       string      `json:"name"`
	Tasks      []Task      `json:"tasks,omitempty"`
	Variables  []Variable  `json:"variables,omitempty"`
	Namespaces []Namespace `json:"namespaces,omitempty"`
}

// Command represents a single command line in a task
type Command struct {
	Line                  string `json:"line"`
	Silent                bool   `json:"silent,omitempty"`
	ContinueOnError       bool   `json:"continue_on_error,omitempty"`
	IsCommandSubstitution bool   `json:"is_command_substitution,omitempty"`
}
