package jsonpatch

import (
	"encoding/json"
	"fmt"
	"strings"
)

type (
	// Operation represents a JSON Patch operation
	// http://jsonpatch.com/#operations
	Operation struct {
		// Name is the name of the operation - this can be one of the following:
		// add, remove, replace, copy, move or test
		Name string `json:"op"`
		// Path is a JSON pointer
		Path string `json:"path"`
		// Value is used by add, replace and test operations as the JSON value to
		// add, replace with or test for
		Value interface{} `json:"value,omitempty"`
		// From is a JSON pointer used by copy and move operations as the
		// pointer to the value to copy or move
		From string `json:"from,omitempty"`
	}

	// JSONPatch represents a JSON patch - a list of Operations to apply to a
	// JSON object
	JSONPatch struct {
		operations []*Operation
	}
)

func Patch(p ...*Operation) []byte {
	bytes, _ := json.Marshal(p)

	return bytes
}

// MarshalJSON returns a JSON byte array representation of the JSONPatch object
func (p *JSONPatch) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.operations)
}

// Add contructs an Operation object to add the value at `path`
func Add(path []string, value interface{}) *Operation {
	return &Operation{
		Name:  "add",
		Path:  escapePath(path),
		Value: value,
	}
}

// Replace constructs an Operation object to replace the value at `path` with
// `value`
func Replace(path []string, value interface{}) *Operation {
	return &Operation{
		Name:  "replace",
		Path:  escapePath(path),
		Value: value,
	}
}

// Remove constructs an Operation object to remove the value at `path`
func Remove(path []string) *Operation {
	return &Operation{
		Name: "remove",
		Path: escapePath(path),
	}
}

// escapePath constructs a JSON pointer string from zero or more strings
// representing keys or array indices can be passed to construct the path. "-"
// can be used to represent the end of an array. Key names are automatically
// escaped.
func escapePath(parts []string) string {
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		escaped = append(escaped, escape(part))
	}
	return fmt.Sprintf("/%s", strings.Join(escaped, "/"))
}

// Escape returns the string formatted for use in a JSON pointer in a JSON
// patch.
// JSON patch requires "~" and "/" characters to be escaped as "~0" and "~1"
// respectively. See http://jsonpatch.com/#json-pointer
func escape(s string) (escaped string) {
	escaped = strings.Replace(s, "~", "~0", -1)
	escaped = strings.Replace(escaped, "/", "~1", -1)

	return
}
