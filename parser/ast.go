package parser

// QuakeFile represents the root of a parsed Quakefile
type QuakeFile struct {
	Tasks     []Task     `json:"tasks"`
	Variables []Variable `json:"variables,omitempty"`
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

// Command represents a single command line in a task
type Command struct {
	Line            string `json:"line"`
	Silent          bool   `json:"silent,omitempty"`
	ContinueOnError bool   `json:"continue_on_error,omitempty"`
}