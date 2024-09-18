package runtime

type RuntimeScope struct {
	Parent    *RuntimeScope
	Symbols   map[string]RuntimeValue
	Constants map[string]RuntimeValue
}

// DeclareSymbol declares a symbol in the current scope and returns the value
func (scope *RuntimeScope) DeclareSymbol(name string, value RuntimeValue, isConstant bool) (RuntimeValue, bool) {
	if _, exists := scope.Symbols[name]; exists {
		return scope.Symbols[name], false
	}

	scope.Symbols[name] = value

	if isConstant {
		scope.Constants[name] = value
	}

	return value, true
}

// AssignSymbol assigns a value to a symbol in the current scope
// If the symbol does not exist, it will check the parent scope
// If the symbol is a constant, it will panic
func (scope *RuntimeScope) AssignSymbol(name string, value RuntimeValue) (RuntimeValue, bool) {
	applicable := scope.ResolveScope(name)

	if _, exists := applicable.Constants[name]; exists {
		panic("Cannot assign to a constant")
	}

	applicable.Symbols[name] = value

	return value, true
}

// GetSymbol returns the value of a symbol in the current scope
// If the symbol does not exist, it will check the parent scope
func (scope *RuntimeScope) GetSymbol(name string) (RuntimeValue, bool) {
	applicable := scope.ResolveScope(name)

	if applicable == nil {
		return RuntimeValue{
			Type:  NoneValue,
			Name:  ValueNames[NoneValue],
			Value: nil,
		}, false
	}

	if _, exists := applicable.Symbols[name]; exists {
		return applicable.Symbols[name], true
	}

	return RuntimeValue{
		Type:  NoneValue,
		Name:  ValueNames[NoneValue],
		Value: nil,
	}, false
}

func (scope *RuntimeScope) ResolveScope(name string) *RuntimeScope {

	if _, exists := scope.Symbols[name]; exists {
		return scope
	}

	if scope.Parent == nil {
		return nil
	}

	return scope.Parent.ResolveScope(name)
}
