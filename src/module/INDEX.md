# module/ — Module resolution and dependency graph

| File | Contents |
|------|----------|
| `module.go` | `Build()` entry point, `ResolvePath()`, `collectExportNames()`, DFS cycle detection, topological sort. Types: `Module`, `Graph`. |
| `module_test.go` | Unit tests for path resolution, graph building, cycle detection, export validation. |
