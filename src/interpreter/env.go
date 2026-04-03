package interpreter

import "fmt"

// Environment holds variable bindings with lexical scoping.
type Environment struct {
	values map[string]binding
	parent *Environment
}

type binding struct {
	value   Value
	isConst bool
}

// NewEnvironment creates a root environment.
func NewEnvironment() *Environment {
	return &Environment{values: make(map[string]binding)}
}

// Child creates a child scope with this environment as parent.
func (e *Environment) Child() *Environment {
	return &Environment{values: make(map[string]binding), parent: e}
}

// Snapshot creates a deep copy of the environment for closure capture.
// Only copies the current scope's bindings, with parent chain shared.
func (e *Environment) Snapshot() *Environment {
	copied := make(map[string]binding, len(e.values))
	for k, v := range e.values {
		copied[k] = binding{value: v.value.DeepCopy(), isConst: v.isConst}
	}
	var parentCopy *Environment
	if e.parent != nil {
		parentCopy = e.parent.Snapshot()
	}
	return &Environment{values: copied, parent: parentCopy}
}

// Define declares a new variable in the current scope.
func (e *Environment) Define(name string, val Value, isConst bool) error {
	if _, exists := e.values[name]; exists {
		return fmt.Errorf("variable '%s' already declared in this scope", name)
	}
	e.values[name] = binding{value: val, isConst: isConst}
	return nil
}

// Get looks up a variable, walking up the scope chain.
func (e *Environment) Get(name string) (Value, error) {
	if b, ok := e.values[name]; ok {
		return b.value, nil
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return MonkNone, fmt.Errorf("undefined variable '%s'", name)
}

// Set assigns to an existing variable, walking up the scope chain.
func (e *Environment) Set(name string, val Value) error {
	if b, ok := e.values[name]; ok {
		if b.isConst {
			return fmt.Errorf("cannot reassign const '%s'", name)
		}
		e.values[name] = binding{value: val, isConst: false}
		return nil
	}
	if e.parent != nil {
		return e.parent.Set(name, val)
	}
	return fmt.Errorf("undefined variable '%s'", name)
}

// IsConst checks if a variable is const (for deep const enforcement).
func (e *Environment) IsConst(name string) bool {
	if b, ok := e.values[name]; ok {
		return b.isConst
	}
	if e.parent != nil {
		return e.parent.IsConst(name)
	}
	return false
}
