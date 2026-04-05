import * as vscode from "vscode";
import { BUILTINS, KEYWORDS, TYPES } from "./builtins";

export class MonkHoverProvider implements vscode.HoverProvider {
  provideHover(
    document: vscode.TextDocument,
    position: vscode.Position
  ): vscode.Hover | undefined {
    const range = document.getWordRangeAtPosition(position, /[a-zA-Z_][a-zA-Z0-9_]*/);
    if (!range) return undefined;

    const word = document.getText(range);

    // Builtin function hover
    const builtin = BUILTINS[word];
    if (builtin) {
      const md = new vscode.MarkdownString();
      md.appendCodeblock(builtin.signature, "monk");
      md.appendMarkdown(`\n\n${builtin.description}`);
      md.appendMarkdown(`\n\n**Returns:** \`${builtin.returns}\``);
      return new vscode.Hover(md, range);
    }

    // Keyword hover
    if (KEYWORDS.includes(word)) {
      const info = KEYWORD_DOCS[word];
      if (info) {
        const md = new vscode.MarkdownString();
        md.appendMarkdown(`**${word}** — ${info}`);
        return new vscode.Hover(md, range);
      }
    }

    // Type hover
    if (TYPES.includes(word)) {
      const info = TYPE_DOCS[word];
      if (info) {
        const md = new vscode.MarkdownString();
        md.appendMarkdown(`**${word}** — ${info}`);
        return new vscode.Hover(md, range);
      }
    }

    return undefined;
  }
}

const KEYWORD_DOCS: Record<string, string> = {
  let: "Declare a mutable variable. Can be reassigned.",
  const: "Declare an immutable constant. Deeply frozen — cannot change the value or its contents.",
  if: "Conditional branch. Braces required.",
  else: "Alternative branch for `if`.",
  for: "Iterate over an array or string. `for item in collection { ... }`",
  in: "Used with `for` loops to iterate over a collection.",
  while: "Loop while condition is truthy. `while condition { ... }`",
  break: "Exit the current loop immediately.",
  continue: "Skip to the next iteration of the current loop.",
  return: "Return a value from a function.",
  guard: "Error handling. `guard result = expr against error { ... }` Catches thrown errors.",
  against: "The catch block of a `guard` expression.",
  throw: "Throw an error value. Caught by the nearest `guard/against`.",
  type: "Declare a named record type or type alias.",
  use: "Import from another module. `use name from \"./path\"`",
  export: "Export a declaration for use by other modules.",
  from: "Specify the module path in a `use` statement.",
  as: "Alias an import. `use name as alias from \"./path\"`",
  and: "Logical AND operator. Short-circuits.",
  or: "Logical OR operator. Short-circuits.",
  not: "Logical NOT operator.",
  is: "Type check operator.",
  true: "Boolean true. Truthy.",
  false: "Boolean false. Falsy.",
  none: "The absence of a value. Falsy.",
};

const TYPE_DOCS: Record<string, string> = {
  int: "64-bit signed integer. Falsy when 0.",
  float: "64-bit IEEE 754 floating point.",
  string: "UTF-8 encoded string. Always truthy (even empty string).",
  boolean: "`true` or `false`.",
  none: "The absence of a value. Always falsy.",
};
