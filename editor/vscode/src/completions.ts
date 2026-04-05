import * as vscode from "vscode";
import { BUILTINS, KEYWORDS, TYPES } from "./builtins";

export class MonkCompletionProvider
  implements vscode.CompletionItemProvider
{
  provideCompletionItems(
    document: vscode.TextDocument,
    position: vscode.Position
  ): vscode.CompletionItem[] {
    const items: vscode.CompletionItem[] = [];

    // Builtins with full docs
    for (const [name, info] of Object.entries(BUILTINS)) {
      const item = new vscode.CompletionItem(
        name,
        vscode.CompletionItemKind.Function
      );
      item.detail = info.signature;
      item.documentation = new vscode.MarkdownString(
        `${info.description}\n\n**Returns:** \`${info.returns}\``
      );
      item.insertText = new vscode.SnippetString(`${name}(\${1})`);
      items.push(item);
    }

    // Keywords with smart snippets
    for (const kw of KEYWORDS) {
      const item = new vscode.CompletionItem(
        kw,
        vscode.CompletionItemKind.Keyword
      );
      item.detail = "keyword";

      if (kw === "if" || kw === "while") {
        item.insertText = new vscode.SnippetString(
          `${kw} \${1:condition} {\n\t\${0}\n}`
        );
      } else if (kw === "for") {
        item.insertText = new vscode.SnippetString(
          `for \${1:item} in \${2:collection} {\n\t\${0}\n}`
        );
      } else if (kw === "guard") {
        item.insertText = new vscode.SnippetString(
          `guard \${1:result} = \${2:expr} against \${3:error} {\n\t\${1} = \${0}\n}`
        );
      } else if (kw === "let") {
        item.insertText = new vscode.SnippetString(
          `let \${1:name} = \${0}`
        );
      } else if (kw === "const") {
        item.insertText = new vscode.SnippetString(
          `const \${1:NAME} = \${0}`
        );
      }

      items.push(item);
    }

    // Types
    for (const t of TYPES) {
      const item = new vscode.CompletionItem(
        t,
        vscode.CompletionItemKind.TypeParameter
      );
      item.detail = "type";
      items.push(item);
    }

    // Array type variants
    for (const t of TYPES.filter((t) => t !== "none")) {
      const item = new vscode.CompletionItem(
        `${t}[]`,
        vscode.CompletionItemKind.TypeParameter
      );
      item.detail = "array type";
      items.push(item);
    }

    // Optional type variants
    for (const t of TYPES.filter((t) => t !== "none")) {
      const item = new vscode.CompletionItem(
        `${t}?`,
        vscode.CompletionItemKind.TypeParameter
      );
      item.detail = "optional type";
      items.push(item);
    }

    // User-defined variables from the document
    const text = document.getText();
    const varPattern = /\b(?:let|const)\s+([a-zA-Z_][a-zA-Z0-9_]*)/g;
    const seen = new Set<string>();
    let match;
    while ((match = varPattern.exec(text)) !== null) {
      const name = match[1];
      if (!seen.has(name) && !BUILTINS[name] && !KEYWORDS.includes(name)) {
        seen.add(name);
        const item = new vscode.CompletionItem(
          name,
          vscode.CompletionItemKind.Variable
        );
        item.detail = "variable";
        items.push(item);
      }
    }

    return items;
  }
}
