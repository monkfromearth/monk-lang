package interpreter

import (
	"fmt"
	"strconv"

	"github.com/monkfromearth/monk-lang/src/syntax"
)

// Control flow signals. These implement error so they propagate through
// the call stack. The evaluator catches them at the appropriate level:
// returnSignal at function call boundaries, breakSignal/continueSignal
// at loop boundaries, throwSignal at guard/against boundaries.
type returnSignal struct{ value Value }
type breakSignal struct{}
type continueSignal struct{}
type throwSignal struct{ value Value }

func (s returnSignal) Error() string   { return "return" }
func (s breakSignal) Error() string    { return "break" }
func (s continueSignal) Error() string { return "continue" }
func (s throwSignal) Error() string    { return fmt.Sprintf("throw: %s", s.value.String()) }

// Eval parses and evaluates a Monk source string. Returns the value of the
// last expression, or MonkNone for empty programs / void statements.
func Eval(source string) (Value, error) {
	prog, err := syntax.Parse(source)
	if err != nil {
		return MonkNone, err
	}

	env := NewEnvironment()
	return evalProgram(prog, env)
}

func evalProgram(prog *syntax.Program, env *Environment) (Value, error) {
	var last Value = MonkNone
	for _, stmt := range prog.Stmts {
		val, err := evalStmt(stmt, env)
		if err != nil {
			return MonkNone, err
		}
		last = val
	}
	return last, nil
}

// --- Statements ---

func evalStmt(stmt syntax.Stmt, env *Environment) (Value, error) {
	switch s := stmt.(type) {
	case *syntax.VarDeclStmt:
		return evalVarDecl(s, env)
	case *syntax.ExprStmt:
		return evalExpr(s.Expr, env)
	case *syntax.BlockStmt:
		return evalBlock(s, env.Child())
	case *syntax.IfStmt:
		return evalIf(s, env)
	case *syntax.WhileStmt:
		return evalWhile(s, env)
	case *syntax.ForStmt:
		return evalFor(s, env)
	case *syntax.ReturnStmt:
		return evalReturn(s, env)
	case *syntax.BreakStmt:
		return MonkNone, breakSignal{}
	case *syntax.ContinueStmt:
		return MonkNone, continueSignal{}
	case *syntax.GuardStmt:
		return evalGuard(s, env)
	default:
		return MonkNone, fmt.Errorf("unknown statement type: %T", stmt)
	}
}

func evalVarDecl(s *syntax.VarDeclStmt, env *Environment) (Value, error) {
	// For function values: snapshot the env (closure captures by copy),
	// then inject the function's own name into the snapshot so it can
	// call itself recursively. This gives us recursion WITHOUT sharing
	// the live environment.
	if fnExpr, isFn := s.Value.(*syntax.FuncExpr); isFn {
		capturedEnv := env.Snapshot()
		val := makeFuncValue(fnExpr, capturedEnv)
		// Inject self-reference for recursion
		capturedEnv.Define(s.Name, val, s.IsConst)
		if err := env.Define(s.Name, val, s.IsConst); err != nil {
			return MonkNone, err
		}
		return MonkNone, nil
	}

	val, err := evalExpr(s.Value, env)
	if err != nil {
		return MonkNone, err
	}
	val = val.DeepCopy() // value semantics
	if err := env.Define(s.Name, val, s.IsConst); err != nil {
		return MonkNone, err
	}
	return MonkNone, nil
}

func evalBlock(block *syntax.BlockStmt, env *Environment) (Value, error) {
	var last Value = MonkNone
	for _, stmt := range block.Stmts {
		val, err := evalStmt(stmt, env)
		if err != nil {
			return MonkNone, err
		}
		last = val
	}
	return last, nil
}

func evalIf(s *syntax.IfStmt, env *Environment) (Value, error) {
	cond, err := evalExpr(s.Condition, env)
	if err != nil {
		return MonkNone, err
	}
	if cond.IsTruthy() {
		return evalBlock(s.Then, env.Child())
	}
	if s.Else != nil {
		switch e := s.Else.(type) {
		case *syntax.BlockStmt:
			return evalBlock(e, env.Child())
		case *syntax.IfStmt:
			return evalIf(e, env)
		}
	}
	return MonkNone, nil
}

func evalWhile(s *syntax.WhileStmt, env *Environment) (Value, error) {
	for {
		cond, err := evalExpr(s.Condition, env)
		if err != nil {
			return MonkNone, err
		}
		if !cond.IsTruthy() {
			break
		}
		_, err = evalBlock(s.Body, env.Child())
		if err != nil {
			if _, ok := err.(breakSignal); ok {
				break
			}
			if _, ok := err.(continueSignal); ok {
				continue
			}
			return MonkNone, err
		}
	}
	return MonkNone, nil
}

func evalFor(s *syntax.ForStmt, env *Environment) (Value, error) {
	iterable, err := evalExpr(s.Iterable, env)
	if err != nil {
		return MonkNone, err
	}

	var items []Value
	switch iterable.Kind {
	case ArrayValue:
		items = iterable.Array
	case StringValue:
		for _, ch := range []byte(iterable.Str) {
			items = append(items, StringVal(string(ch)))
		}
	default:
		return MonkNone, fmt.Errorf("cannot iterate over %s", iterable.TypeName())
	}

	for _, item := range items {
		loopEnv := env.Child()
		// Loop variable is const
		if err := loopEnv.Define(s.VarName, item.DeepCopy(), true); err != nil {
			return MonkNone, err
		}
		_, err := evalBlock(s.Body, loopEnv)
		if err != nil {
			if _, ok := err.(breakSignal); ok {
				break
			}
			if _, ok := err.(continueSignal); ok {
				continue
			}
			return MonkNone, err
		}
	}
	return MonkNone, nil
}

func evalReturn(s *syntax.ReturnStmt, env *Environment) (Value, error) {
	if s.Value == nil {
		return MonkNone, returnSignal{value: MonkNone}
	}
	val, err := evalExpr(s.Value, env)
	if err != nil {
		return MonkNone, err
	}
	return MonkNone, returnSignal{value: val}
}

func evalGuard(s *syntax.GuardStmt, env *Environment) (Value, error) {
	val, err := evalExpr(s.Expr, env)

	if err != nil {
		// Check if it's a throw
		if ts, ok := err.(throwSignal); ok {
			// Error path: run the against block
			env.Define(s.VarName, MonkNone, false)
			againstEnv := env.Child()
			againstEnv.Define(s.ErrorName, ts.value, true)
			_, blockErr := evalBlock(s.Against, againstEnv)
			if blockErr != nil {
				return MonkNone, blockErr
			}
			// Variable might have been assigned inside the block
			result, _ := env.Get(s.VarName)
			return result, nil
		}
		return MonkNone, err
	}

	// Success path
	env.Define(s.VarName, val, false)
	return MonkNone, nil
}

// --- Expressions ---

func evalExpr(expr syntax.Expr, env *Environment) (Value, error) {
	switch e := expr.(type) {
	case *syntax.NumberExpr:
		return evalNumber(e)
	case *syntax.StringExpr:
		return StringVal(e.Value), nil
	case *syntax.TemplateExpr:
		return StringVal(e.Value), nil
	case *syntax.BoolExpr:
		return BoolVal(e.Value), nil
	case *syntax.NoneExpr:
		return MonkNone, nil
	case *syntax.IdentExpr:
		return env.Get(e.Name)
	case *syntax.UnaryExpr:
		return evalUnary(e, env)
	case *syntax.BinaryExpr:
		return evalBinary(e, env)
	case *syntax.AssignExpr:
		return evalAssign(e, env)
	case *syntax.CallExpr:
		return evalCall(e, env)
	case *syntax.IndexExpr:
		return evalIndex(e, env)
	case *syntax.PropertyExpr:
		return evalProperty(e, env)
	case *syntax.ArrayExpr:
		return evalArray(e, env)
	case *syntax.RecordExpr:
		return evalRecord(e, env)
	case *syntax.FuncExpr:
		return evalFuncExpr(e, env)
	case *syntax.ThrowExpr:
		return evalThrow(e, env)
	default:
		return MonkNone, fmt.Errorf("unknown expression type: %T", expr)
	}
}

func evalNumber(e *syntax.NumberExpr) (Value, error) {
	if e.IsInt {
		n, err := strconv.ParseInt(e.Value, 0, 64)
		if err != nil {
			return MonkNone, fmt.Errorf("invalid integer: %s", e.Value)
		}
		return IntVal(n), nil
	}
	f, err := strconv.ParseFloat(e.Value, 64)
	if err != nil {
		return MonkNone, fmt.Errorf("invalid float: %s", e.Value)
	}
	return FloatVal(f), nil
}

func evalUnary(e *syntax.UnaryExpr, env *Environment) (Value, error) {
	operand, err := evalExpr(e.Operand, env)
	if err != nil {
		return MonkNone, err
	}

	switch e.Op {
	case syntax.Minus:
		switch operand.Kind {
		case IntValue:
			return IntVal(-operand.Int), nil
		case FloatValue:
			return FloatVal(-operand.Float), nil
		default:
			return MonkNone, fmt.Errorf("cannot negate %s", operand.TypeName())
		}

	case syntax.Not, syntax.Bang:
		return BoolVal(!operand.IsTruthy()), nil

	case syntax.Tilde:
		if operand.Kind != IntValue {
			return MonkNone, fmt.Errorf("bitwise NOT requires int, got %s", operand.TypeName())
		}
		return IntVal(^operand.Int), nil

	default:
		return MonkNone, fmt.Errorf("unknown unary operator: %s", e.Op)
	}
}

func evalBinary(e *syntax.BinaryExpr, env *Environment) (Value, error) {
	// Short-circuit for logical operators
	if e.Op == syntax.And || e.Op == syntax.AmpAmp {
		left, err := evalExpr(e.Left, env)
		if err != nil {
			return MonkNone, err
		}
		if !left.IsTruthy() {
			return MonkFalse, nil
		}
		right, err := evalExpr(e.Right, env)
		if err != nil {
			return MonkNone, err
		}
		return BoolVal(right.IsTruthy()), nil
	}

	if e.Op == syntax.Or || e.Op == syntax.PipePipe {
		left, err := evalExpr(e.Left, env)
		if err != nil {
			return MonkNone, err
		}
		if left.IsTruthy() {
			return MonkTrue, nil
		}
		right, err := evalExpr(e.Right, env)
		if err != nil {
			return MonkNone, err
		}
		return BoolVal(right.IsTruthy()), nil
	}

	left, err := evalExpr(e.Left, env)
	if err != nil {
		return MonkNone, err
	}
	right, err := evalExpr(e.Right, env)
	if err != nil {
		return MonkNone, err
	}

	// String concatenation
	if e.Op == syntax.Plus && left.Kind == StringValue && right.Kind == StringValue {
		return StringVal(left.Str + right.Str), nil
	}

	// Equality: same-type primitives only
	if e.Op == syntax.EqualEqual || e.Op == syntax.Is {
		return evalEquality(left, right, true)
	}
	if e.Op == syntax.BangEqual {
		return evalEquality(left, right, false)
	}

	// Ordering: numbers and strings
	if isOrderOp(e.Op) {
		return evalOrdering(left, right, e.Op)
	}

	// Bitwise: int only
	if isBitwiseOp(e.Op) {
		return evalBitwise(left, right, e.Op)
	}

	// Arithmetic: numeric only
	return evalArithmetic(left, right, e.Op)
}

func evalEquality(left, right Value, isEqual bool) (Value, error) {
	// none == none
	if left.Kind == NoneValue && right.Kind == NoneValue {
		return BoolVal(isEqual), nil
	}
	// none vs non-none
	if left.Kind == NoneValue || right.Kind == NoneValue {
		return BoolVal(!isEqual), nil
	}
	// Cross-type comparison is an error
	if left.Kind != right.Kind {
		return MonkNone, fmt.Errorf("cannot compare %s and %s", left.TypeName(), right.TypeName())
	}
	// Same type comparison
	var eq bool
	switch left.Kind {
	case IntValue:
		eq = left.Int == right.Int
	case FloatValue:
		eq = left.Float == right.Float
	case StringValue:
		eq = left.Str == right.Str
	case BoolValue:
		eq = left.Bool == right.Bool
	default:
		return MonkNone, fmt.Errorf("cannot compare %s values", left.TypeName())
	}
	if !isEqual {
		eq = !eq
	}
	return BoolVal(eq), nil
}

func evalOrdering(left, right Value, op syntax.TokenKind) (Value, error) {
	if left.Kind == NoneValue || right.Kind == NoneValue {
		return MonkNone, fmt.Errorf("cannot order none")
	}
	if left.Kind != right.Kind {
		return MonkNone, fmt.Errorf("cannot compare %s and %s", left.TypeName(), right.TypeName())
	}

	var result bool
	switch left.Kind {
	case IntValue:
		switch op {
		case syntax.Less:
			result = left.Int < right.Int
		case syntax.Greater:
			result = left.Int > right.Int
		case syntax.LessEqual:
			result = left.Int <= right.Int
		case syntax.GreaterEqual:
			result = left.Int >= right.Int
		}
	case FloatValue:
		switch op {
		case syntax.Less:
			result = left.Float < right.Float
		case syntax.Greater:
			result = left.Float > right.Float
		case syntax.LessEqual:
			result = left.Float <= right.Float
		case syntax.GreaterEqual:
			result = left.Float >= right.Float
		}
	case StringValue:
		switch op {
		case syntax.Less:
			result = left.Str < right.Str
		case syntax.Greater:
			result = left.Str > right.Str
		case syntax.LessEqual:
			result = left.Str <= right.Str
		case syntax.GreaterEqual:
			result = left.Str >= right.Str
		}
	default:
		return MonkNone, fmt.Errorf("cannot order %s values", left.TypeName())
	}
	return BoolVal(result), nil
}

func evalBitwise(left, right Value, op syntax.TokenKind) (Value, error) {
	if left.Kind != IntValue || right.Kind != IntValue {
		return MonkNone, fmt.Errorf("bitwise operators require int, got %s and %s", left.TypeName(), right.TypeName())
	}
	switch op {
	case syntax.Amp:
		return IntVal(left.Int & right.Int), nil
	case syntax.Pipe:
		return IntVal(left.Int | right.Int), nil
	case syntax.Caret:
		return IntVal(left.Int ^ right.Int), nil
	case syntax.ShiftLeft:
		return IntVal(left.Int << uint(right.Int)), nil
	case syntax.ShiftRight:
		return IntVal(left.Int >> uint(right.Int)), nil
	default:
		return MonkNone, fmt.Errorf("unknown bitwise operator: %s", op)
	}
}

func evalArithmetic(left, right Value, op syntax.TokenKind) (Value, error) {
	// Promote to float if either side is float
	if left.Kind == FloatValue || right.Kind == FloatValue {
		lf := toFloat(left)
		rf := toFloat(right)
		if lf == nil || rf == nil {
			return MonkNone, fmt.Errorf("cannot apply %s to %s and %s", op, left.TypeName(), right.TypeName())
		}
		switch op {
		case syntax.Plus:
			return FloatVal(*lf + *rf), nil
		case syntax.Minus:
			return FloatVal(*lf - *rf), nil
		case syntax.Star:
			return FloatVal(*lf * *rf), nil
		case syntax.Slash:
			if *rf == 0 {
				return MonkNone, fmt.Errorf("division by zero")
			}
			return FloatVal(*lf / *rf), nil
		case syntax.Percent:
			return MonkNone, fmt.Errorf("modulo not supported on floats")
		}
	}

	// Int arithmetic
	if left.Kind != IntValue || right.Kind != IntValue {
		return MonkNone, fmt.Errorf("cannot apply %s to %s and %s", op, left.TypeName(), right.TypeName())
	}

	switch op {
	case syntax.Plus:
		return IntVal(left.Int + right.Int), nil
	case syntax.Minus:
		return IntVal(left.Int - right.Int), nil
	case syntax.Star:
		return IntVal(left.Int * right.Int), nil
	case syntax.Slash:
		if right.Int == 0 {
			return MonkNone, fmt.Errorf("division by zero")
		}
		return IntVal(left.Int / right.Int), nil
	case syntax.Percent:
		if right.Int == 0 {
			return MonkNone, fmt.Errorf("modulo by zero")
		}
		return IntVal(left.Int % right.Int), nil
	default:
		return MonkNone, fmt.Errorf("unknown arithmetic operator: %s", op)
	}
}

func evalAssign(e *syntax.AssignExpr, env *Environment) (Value, error) {
	val, err := evalExpr(e.Value, env)
	if err != nil {
		return MonkNone, err
	}

	switch target := e.Target.(type) {
	case *syntax.IdentExpr:
		if e.Op != syntax.Equal {
			// Compound assignment: get current value first
			current, err := env.Get(target.Name)
			if err != nil {
				return MonkNone, err
			}
			val, err = evalCompound(current, val, e.Op)
			if err != nil {
				return MonkNone, err
			}
		}
		val = val.DeepCopy()
		if err := env.Set(target.Name, val); err != nil {
			return MonkNone, err
		}
		return val, nil

	case *syntax.IndexExpr:
		return evalIndexAssign(target, val, e.Op, env)

	case *syntax.PropertyExpr:
		return evalPropertyAssign(target, val, e.Op, env)

	default:
		return MonkNone, fmt.Errorf("invalid assignment target: %T", e.Target)
	}
}

func evalCompound(current, operand Value, op syntax.TokenKind) (Value, error) {
	switch op {
	case syntax.PlusEqual:
		return evalArithmetic(current, operand, syntax.Plus)
	case syntax.MinusEqual:
		return evalArithmetic(current, operand, syntax.Minus)
	case syntax.StarEqual:
		return evalArithmetic(current, operand, syntax.Star)
	case syntax.SlashEqual:
		return evalArithmetic(current, operand, syntax.Slash)
	case syntax.PercentEqual:
		return evalArithmetic(current, operand, syntax.Percent)
	default:
		return MonkNone, fmt.Errorf("unknown compound operator: %s", op)
	}
}

func evalCall(e *syntax.CallExpr, env *Environment) (Value, error) {
	callee, err := evalExpr(e.Callee, env)
	if err != nil {
		return MonkNone, err
	}

	if callee.Kind != FuncValue {
		return MonkNone, fmt.Errorf("cannot call %s", callee.TypeName())
	}

	fn := callee.Func

	// Evaluate arguments (copies — value semantics)
	args := make([]Value, len(e.Args))
	for i, arg := range e.Args {
		val, err := evalExpr(arg, env)
		if err != nil {
			return MonkNone, err
		}
		args[i] = val.DeepCopy()
	}

	// Create function scope from closure's captured environment
	fnEnv := fn.Env.Child()
	params := fn.Params
	for i, param := range params {
		if i < len(args) {
			fnEnv.Define(param, args[i], false)
		}
	}

	// Execute body
	body := fn.Body.(*syntax.BlockStmt)
	_, err = evalBlock(body, fnEnv)
	if err != nil {
		if rs, ok := err.(returnSignal); ok {
			return rs.value, nil
		}
		return MonkNone, err
	}

	return MonkNone, nil
}

func evalIndex(e *syntax.IndexExpr, env *Environment) (Value, error) {
	object, err := evalExpr(e.Object, env)
	if err != nil {
		return MonkNone, err
	}

	index, err := evalExpr(e.Index, env)
	if err != nil {
		return MonkNone, err
	}

	if index.Kind != IntValue {
		return MonkNone, fmt.Errorf("index must be int, got %s", index.TypeName())
	}
	idx := index.Int

	switch object.Kind {
	case ArrayValue:
		if idx < 0 || idx >= int64(len(object.Array)) {
			return MonkNone, nil // graceful on reads
		}
		return object.Array[idx].DeepCopy(), nil

	case StringValue:
		if idx < 0 || idx >= int64(len(object.Str)) {
			return MonkNone, nil // graceful on reads
		}
		return StringVal(string(object.Str[idx])), nil

	default:
		return MonkNone, fmt.Errorf("cannot index %s", object.TypeName())
	}
}

func evalIndexAssign(target *syntax.IndexExpr, val Value, op syntax.TokenKind, env *Environment) (Value, error) {
	// Get the variable name for const check
	ident, ok := target.Object.(*syntax.IdentExpr)
	if !ok {
		return MonkNone, fmt.Errorf("cannot assign to index of expression")
	}

	if env.IsConst(ident.Name) {
		return MonkNone, fmt.Errorf("cannot mutate const '%s'", ident.Name)
	}

	object, err := env.Get(ident.Name)
	if err != nil {
		return MonkNone, err
	}

	index, err := evalExpr(target.Index, env)
	if err != nil {
		return MonkNone, err
	}

	if index.Kind != IntValue {
		return MonkNone, fmt.Errorf("index must be int, got %s", index.TypeName())
	}
	idx := index.Int

	if object.Kind != ArrayValue {
		return MonkNone, fmt.Errorf("cannot index-assign to %s", object.TypeName())
	}

	if idx < 0 || idx >= int64(len(object.Array)) {
		return MonkNone, fmt.Errorf("index %d out of bounds (length %d)", idx, len(object.Array))
	}

	if op != syntax.Equal {
		current := object.Array[idx]
		val, err = evalCompound(current, val, op)
		if err != nil {
			return MonkNone, err
		}
	}

	object.Array[idx] = val.DeepCopy()
	env.Set(ident.Name, object)
	return val, nil
}

func evalProperty(e *syntax.PropertyExpr, env *Environment) (Value, error) {
	object, err := evalExpr(e.Object, env)
	if err != nil {
		return MonkNone, err
	}

	if object.Kind != RecordValue {
		return MonkNone, fmt.Errorf("cannot access property of %s", object.TypeName())
	}

	for _, entry := range object.Record {
		if entry.Key == e.Property {
			return entry.Value.DeepCopy(), nil
		}
	}

	// Untyped record: missing field returns none (graceful on reads)
	return MonkNone, nil
}

func evalPropertyAssign(target *syntax.PropertyExpr, val Value, op syntax.TokenKind, env *Environment) (Value, error) {
	ident, ok := target.Object.(*syntax.IdentExpr)
	if !ok {
		return MonkNone, fmt.Errorf("cannot assign to property of expression")
	}

	if env.IsConst(ident.Name) {
		return MonkNone, fmt.Errorf("cannot mutate const '%s'", ident.Name)
	}

	object, err := env.Get(ident.Name)
	if err != nil {
		return MonkNone, err
	}

	if object.Kind != RecordValue {
		return MonkNone, fmt.Errorf("cannot set property on %s", object.TypeName())
	}

	// Find existing field (records have fixed shape — no adding new fields)
	found := false
	for i, entry := range object.Record {
		if entry.Key == target.Property {
			if op != syntax.Equal {
				val, err = evalCompound(entry.Value, val, op)
				if err != nil {
					return MonkNone, err
				}
			}
			object.Record[i].Value = val.DeepCopy()
			found = true
			break
		}
	}

	if !found {
		return MonkNone, fmt.Errorf("record has no field '%s'", target.Property)
	}

	env.Set(ident.Name, object)
	return val, nil
}

func evalArray(e *syntax.ArrayExpr, env *Environment) (Value, error) {
	elems := make([]Value, len(e.Elements))
	for i, elem := range e.Elements {
		val, err := evalExpr(elem, env)
		if err != nil {
			return MonkNone, err
		}
		elems[i] = val
	}
	return ArrayVal(elems), nil
}

func evalRecord(e *syntax.RecordExpr, env *Environment) (Value, error) {
	entries := make([]RecordEntry, len(e.Fields))
	for i, field := range e.Fields {
		val, err := evalExpr(field.Value, env)
		if err != nil {
			return MonkNone, err
		}
		entries[i] = RecordEntry{Key: field.Key, Value: val}
	}
	return RecordVal(entries), nil
}

func evalFuncExpr(e *syntax.FuncExpr, env *Environment) (Value, error) {
	// Capture environment snapshot (closure captures by copy)
	return makeFuncValue(e, env.Snapshot()), nil
}

func makeFuncValue(e *syntax.FuncExpr, capturedEnv *Environment) Value {
	params := make([]string, len(e.Params))
	for i, p := range e.Params {
		params[i] = p.Name
	}
	return Value{
		Kind: FuncValue,
		Func: &FuncDef{
			Params: params,
			Env:    capturedEnv,
			Body:   e.Body,
		},
	}
}

func evalThrow(e *syntax.ThrowExpr, env *Environment) (Value, error) {
	val, err := evalExpr(e.Value, env)
	if err != nil {
		return MonkNone, err
	}
	return MonkNone, throwSignal{value: val}
}

// --- Helpers ---

func toFloat(v Value) *float64 {
	switch v.Kind {
	case IntValue:
		f := float64(v.Int)
		return &f
	case FloatValue:
		return &v.Float
	default:
		return nil
	}
}

func isOrderOp(op syntax.TokenKind) bool {
	return op == syntax.Less || op == syntax.Greater || op == syntax.LessEqual || op == syntax.GreaterEqual
}

func isBitwiseOp(op syntax.TokenKind) bool {
	return op == syntax.Amp || op == syntax.Pipe || op == syntax.Caret || op == syntax.ShiftLeft || op == syntax.ShiftRight
}
