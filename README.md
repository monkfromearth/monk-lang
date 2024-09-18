# Monk Lang

A Minimalist, Readable, and Performant Programming Language

### **Design Goals:**
- **Performance**: Powered by Go-like performance characteristics.
- **Readability**: Focusing on the simplicity and readability of Python.
- **Flexibility**: Borrowing the flexible import/export model from JavaScript.
- **Type System**: Light, flexible typing like Go, with the option to declare types when necessary (similar to TypeScript but less complex).
- **Borrowing Model**: Inspired by Rust, but simpler and integrated into the language without needing a `mut` keyword.
- **Minimal Syntax**: No semicolons, no `func` keyword for functions, clean and concise.

---

## **Syntax:**

### 1. **Functions**
- Functions are declared without the `func` keyword. Instead, they are assigned to variables using an arrow function-like syntax.
- **Types are optional** when declaring variables, but can be explicitly declared for parameters and return types.
  
- **Example**:
  ```monk
  add = (a int, b int) int {
      return a + b
  }
  ```

### 2. **Variables: `let` and `const`**
- **`const`**: Declares immutable variables.
- **`let`**: Declares mutable variables.

- **Examples**:
  ```monk
  const pi = 3.14159
  let counter = 0
  counter = counter + 1  # Mutable, so it can be changed
  ```

### 3. **Imports and Exports**
- **`use`** is used to import modules, and **`export`** is used to explicitly define what is exported from a module.
  
- **Example**:
  ```monk
  use { math } from "./math"
  
  math.random()
  ```

  ```monk
  export const random = () {
      return 42
  }
  ```

### 4. **Borrowing Model** (Inspired by Rust but Simpler)
- **Immutable references**: Variables are immutable by default (`const`). When passed as references (`ref`), they are immutable unless declared with `let`.
- **Mutable references**: Variables declared with `let` can be passed as mutable references.

- **Example (Immutable Reference)**:
  ```monk
  const x = 42
  y ref int = &x  # Borrowing x as an immutable reference
  ```

- **Example (Mutable Reference)**:
  ```monk
  let z = 10
  w ref int = &z  # Borrowing z as a mutable reference
  w = 20
  ```

- **Function with Borrowing**:
  ```monk
  increment = (x ref int) {
      show(x + 1)  # Immutable reference, read-only
  }

  update = (x ref int) {
      x = x + 1  # Mutable reference, can modify
  }

  const num = 5
  increment(&num)

  let counter = 10
  update(&counter)  # Mutable borrowing
  ```

### 5. **Error Handling**
- Similar to Go, error handling is explicit and done via return values, avoiding exceptions.
  
- **Example**:
  ```monk
  divide = (a int, b int) (int, string) {
      if b == 0 {
          return (0, "Cannot divide by zero")
      }
      return (a / b, "")
  }

  result, error = divide(10, 0)
  if error != "" {
      show("Error: " + error)
  } else {
      show("Result: " + result)
  }
  ```

### 6. **Ownership and Borrowing**
- **Ownership Transfer**: Passing a variable directly to a function transfers ownership.
- **Borrowing**: Use `&` to pass a reference to avoid ownership transfer.

- **Example (Ownership Transfer)**:
  ```monk
  consume = (x int) {
      show(x)  # Ownership is transferred, the caller cannot use x afterward
  }

  let num = 10
  consume(num)  # After this, num is no longer accessible
  ```

- **Example (Borrowing)**:
  ```monk
  use_borrow = (x ref int) {
      show(x)  # x is borrowed, so the caller retains ownership
  }

  let num = 10
  use_borrow(&num)  # Borrowing without losing ownership
  ```

### 7. **Type System**
- Types are optional for variable declarations (inferred by default), but can be explicitly declared for parameters and return types in functions.
- **Dynamic typing** (like Python) is possible, but static typing can be enforced when necessary.

- **Example**:
  ```monk
  let message = "Hello"
  const number int = 42
  ```

### 8. **Comments**
- Single-line comments use `//`.

- **Example**:
  ```monk
  // This is a comment
  ```

## Features

**Monk Lang** is a new programming language designed to combine performance, readability, and flexibility while maintaining simplicity. Here's a comprehensive overview based on the decisions made:

### **1. Language Semantics and Features**

**1.1 Syntax and Readability**
- **Operators**: Concise syntax for operators, similar to other languages.
- **Control Flow**: Includes standard constructs (`if`, `for`, `while`) with enhanced flexibility.
- **Error Handling**: Explicit error handling using return values and a built-in error type, avoiding exceptions.

**1.2 Type System**
- **Typing**: Optional typing for variables; explicit type declarations for function parameters and return types.
- **Dynamic Typing**: Supported, akin to Python.
- **Static Typing**: Enforced when necessary.
- **Generics**: Not included initially to keep the language simple.

**1.3 Memory Management**
- **Borrowing Model**: Simplified borrowing model similar to Rust.
  - **Immutable References**: Default.
  - **Mutable References**: Explicitly declared with `let`.
- **No Garbage Collection**: To maintain simplicity and performance.

**1.4 Concurrency and Parallelism**
- **Async/Await**: Syntax similar to JavaScript for asynchronous operations.
- **Go Routine Model**: Async functions are supported, but no immediate implementation of goroutines.

### **2. Standard Library and Built-in Functions**

**2.1 Core Libraries**
- **Data Structures and I/O**: Basic core library including essential data structures and I/O functionalities.
- **Utility Libraries**: Collections, algorithms, and utilities are planned but not yet implemented.

### **3. Tooling and Ecosystem**

**3.1 Compiler/Interpreter**
- **Interpreter**: An interpreter is available to run Monk Lang code.

**3.2 IDE Support and Build Tools**
- **Package Management**: Similar to npm, for managing libraries and dependencies.
- **Build Tools**: Provided for compiling and managing Monk Lang projects.

### **4. Language Integration and Interoperability**

**4.1 Foreign Function Interface (FFI)**
- **Not Yet Implemented**: Future consideration for integrating with other languages.

**4.2 Integration with Existing Ecosystems**
- **Not Yet Implemented**: Future consideration for compatibility with existing technologies.

### **5. Security and Safety**

**5.1 Memory and Type Safety**
- **Type System**: Designed to be light, flexible, and easy to use.
- **Borrowing Model**: Ensures memory safety without garbage collection.

**5.2 Security Features**
- **Sandboxing and Data Security**: Not yet implemented, but planned for future development.

### **6. Other Considerations**

**6.1 Syntax and Usability**
- **Readability**: Ensuring a clear and consistent syntax.
- **Error Messages**: Providing informative and helpful error messages.
- **Documentation**: Comprehensive documentation with syntax, semantics, and examples.

**6.2 Performance Optimization**
- **Execution Speed**: Focus on optimizing performance, particularly in critical areas.
- **Memory Usage**: Efficient memory management with the borrowing model.

**6.3 Testing and Debugging**
- **Testing Framework**: Guidelines for testing Monk Lang code, including unit and integration tests.
- **Debugging Tools**: Tools for debugging, including a REPL and logging capabilities.

**6.4 Community and Ecosystem**
- **Community Involvement**: Engaging with the developer community through forums and contributions.
- **Ecosystem Growth**: Building an ecosystem with community-driven libraries and tools.

**6.5 Versioning and Compatibility**
- **Versioning Strategy**: Handling updates and versioning to ensure backward compatibility.
- **Deprecation Policy**: Guidelines for deprecating features and migrating legacy code.

**6.6 Internationalization and Localization**
- **Language Support**: Future consideration for internationalization and localization.

**6.7 Security Considerations**
- **Code Injection**: Safeguards against common security issues like code injection.
- **Data Protection**: Planning for secure handling of sensitive data.

**6.8 Compliance and Licensing**
- **Licensing**: Deciding on the licensing model and ensuring compliance with relevant standards.
- **Compliance**: Meeting standards and regulations for commercial and sensitive use.

**6.9 User Experience**
- **Onboarding**: Smooth onboarding with tutorials and guides.
- **Feedback Mechanism**: A system for users to provide feedback and report issues.

**6.10 Long-Term Vision**
- **Future Features**: Planning for potential advanced features and a development roadmap.
- **Roadmap**: Creating a clear roadmap for language development and feature releases.


**Monk Lang** aims to be a performant, readable, and flexible programming language with a focus on simplicity. It combines elements from languages like Go, Python, and JavaScript while avoiding complexity through careful design decisions. The language will continue to evolve, with future enhancements planned for integration, security, and ecosystem growth.


## **Summary of Monk Lang Features:**
- **No Semicolons**: Minimal syntax.
- **No `func` Keyword**: Functions are assigned using an arrow-like syntax to variables.
- **Imports/Exports**: Using `use` and `export` for clear module boundaries.
- **Borrowing Model**: References (`ref`) with `let`/`const` determining mutability.
- **Error Handling**: Explicit return of errors, similar to Go.
- **Ownership and Borrowing**: Control over whether a variable is borrowed or ownership is transferred.
- **Optional Typing**: Types can be inferred, but can also be explicitly declared where needed.
- **Performance-Oriented**: Designed for performance with Go as the backend.
  
Monk Lang will be simple, easy to read, and efficient, focusing on clear, concise syntax with powerful borrowing and error-handling mechanisms while maintaining flexibility in typing and imports.