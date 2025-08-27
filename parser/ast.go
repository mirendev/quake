package parser

import (
	"encoding/json"
	"fmt"
)

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
	Elements        []CommandElement `json:"elements"`
	Silent          bool             `json:"silent,omitempty"`
	ContinueOnError bool             `json:"continue_on_error,omitempty"`
}

// CommandElement represents a part of a command
type CommandElement interface {
	commandElement()
}

// StringElement represents a literal string in a command
type StringElement struct {
	Value string `json:"value"`
}

func (StringElement) commandElement() {}

// BacktickElement represents a command substitution
type BacktickElement struct {
	Command string `json:"command"`
}

func (BacktickElement) commandElement() {}

// ExpressionElement represents an expression like {{expr}}
type ExpressionElement struct {
	Expression string `json:"expression"`
}

func (ExpressionElement) commandElement() {}

// VariableElement represents a variable reference like $VAR
type VariableElement struct {
	Name string `json:"name"`
}

func (VariableElement) commandElement() {}

// MarshalJSON for Command to handle the interface slice
func (c Command) MarshalJSON() ([]byte, error) {
	// Create concrete types with type tags for marshaling
	elements := make([]any, len(c.Elements))
	for i, elem := range c.Elements {
		switch e := elem.(type) {
		case StringElement:
			elements[i] = struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			}{"string", e.Value}
		case BacktickElement:
			elements[i] = struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			}{"backtick", e.Command}
		case ExpressionElement:
			elements[i] = struct {
				Type       string `json:"type"`
				Expression string `json:"expression"`
			}{"expression", e.Expression}
		case VariableElement:
			elements[i] = struct {
				Type string `json:"type"`
				Name string `json:"name"`
			}{"variable", e.Name}
		default:
			return nil, fmt.Errorf("unknown command element type: %T", e)
		}
	}

	return json.Marshal(struct {
		Elements        []any `json:"elements"`
		Silent          bool  `json:"silent,omitempty"`
		ContinueOnError bool  `json:"continue_on_error,omitempty"`
	}{
		Elements:        elements,
		Silent:          c.Silent,
		ContinueOnError: c.ContinueOnError,
	})
}

// UnmarshalJSON for Command to handle the interface slice
func (c *Command) UnmarshalJSON(data []byte) error {
	var temp struct {
		Elements        []json.RawMessage `json:"elements"`
		Silent          bool              `json:"silent,omitempty"`
		ContinueOnError bool              `json:"continue_on_error,omitempty"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	c.Silent = temp.Silent
	c.ContinueOnError = temp.ContinueOnError
	c.Elements = make([]CommandElement, 0, len(temp.Elements))

	for _, raw := range temp.Elements {
		var typeCheck struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeCheck); err != nil {
			return err
		}

		switch typeCheck.Type {
		case "string":
			var elem StringElement
			if err := json.Unmarshal(raw, &elem); err != nil {
				return err
			}
			c.Elements = append(c.Elements, elem)
		case "backtick":
			var elem BacktickElement
			if err := json.Unmarshal(raw, &elem); err != nil {
				return err
			}
			c.Elements = append(c.Elements, elem)
		case "expression":
			var elem ExpressionElement
			if err := json.Unmarshal(raw, &elem); err != nil {
				return err
			}
			c.Elements = append(c.Elements, elem)
		case "variable":
			var elem VariableElement
			if err := json.Unmarshal(raw, &elem); err != nil {
				return err
			}
			c.Elements = append(c.Elements, elem)
		default:
			return fmt.Errorf("unknown command element type: %s", typeCheck.Type)
		}
	}

	return nil
}
