import * as vscode from "vscode";
import { MonkCompletionProvider } from "./completions";
import { MonkHoverProvider } from "./hover";

export function activate(context: vscode.ExtensionContext) {
  context.subscriptions.push(
    vscode.languages.registerCompletionItemProvider(
      "monk",
      new MonkCompletionProvider(),
      ".",
      "("
    ),
    vscode.languages.registerHoverProvider("monk", new MonkHoverProvider())
  );
}

export function deactivate() {}
