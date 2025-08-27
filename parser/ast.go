package parser

// QuakeFile represents the root of a parsed Quakefile
type QuakeFile struct {
	Tasks         []Task      `json:"tasks"`
	Namespaces    []Namespace `json:"namespaces,omitempty"`
	Variables     []Variable  `json:"variables,omitempty"`
	FileNamespace string      `json:"file_namespace,omitempty"`
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
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Namespace represents a namespace block containing tasks and nested namespaces
type Namespace struct {
	Name       string      `json:"name"`
	Tasks      []Task      `json:"tasks,omitempty"`
	Namespaces []Namespace `json:"namespaces,omitempty"`
}

// Command represents a single command line in a task
type Command struct {
	Line            string `json:"line"`
	Silent          bool   `json:"silent,omitempty"`
	ContinueOnError bool   `json:"continue_on_error,omitempty"`
}
