package lexer

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

func Tokenize(input string) []Token {
	output := []Token{}

	line := 1
	column := 0

	length := len(input)

	i := 0

	for i < length {
		// Decode the next rune starting at position i
		char, size := utf8.DecodeRuneInString(input[i:])

		column++ // Increment column as we process each character

		if char == '\n' {
			line++     // New line encountered, increment line count
			column = 0 // Reset column to 0
			// Move i forward by the number of bytes the rune occupies
			i += size
			output = append(output, Token{Kind: NewlineToken, Value: string(char), Line: line, Column: column})
			continue
		}

		// Skip spaces and tabs
		if char == ' ' || char == '\t' || char == '\r' {
			// Move i forward by the number of bytes the rune occupies
			i += size
			continue
		}

		// Handle single-line comment starting with "//"
		if char == '/' && i+1 < len(input) {
			nextChar, _ := utf8.DecodeRuneInString(input[i+1:])
			if nextChar == '/' {
				// Move i forward to skip the '//' and continue to the end of the line
				i += size // Skip first '/'
				i += size // Skip second '/'
				for i < length {
					char, size = utf8.DecodeRuneInString(input[i:])
					i += size
					// If we encounter a newline, the comment ends
					if char == '\n' {
						line++
						column = 0
						break
					}
				}
				continue // Skip processing further tokens for this comment
			}
		}

		if char == '(' {
			output = append(output, Token{Kind: OpenParenthesisToken, Value: string(char), Line: line, Column: column})
		} else if char == ')' {
			output = append(output, Token{Kind: CloseParenthesisToken, Value: string(char), Line: line, Column: column})
		} else if char == '{' {
			output = append(output, Token{Kind: OpenCurlyBraceToken, Value: string(char), Line: line, Column: column})
		} else if char == '}' {
			output = append(output, Token{Kind: CloseCurlyBraceToken, Value: string(char), Line: line, Column: column})
		} else if char == '[' {
			output = append(output, Token{Kind: OpenBracketToken, Value: string(char), Line: line, Column: column})
		} else if char == ']' {
			output = append(output, Token{Kind: CloseBracketToken, Value: string(char), Line: line, Column: column})
		} else if char == ':' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == ':' {
					// Process the `+=` token
					output = append(output, Token{Kind: ColonColonToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			output = append(output, Token{Kind: ColonToken, Value: string(char), Line: line, Column: column})
		} else if char == ',' {
			output = append(output, Token{Kind: CommaToken, Value: string(char), Line: line, Column: column})
		} else if char == '.' {
			output = append(output, Token{Kind: DotToken, Value: string(char), Line: line, Column: column})
		} else if char == '+' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '=' {
					// Process the `+=` token
					output = append(output, Token{Kind: PlusEqualsToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			// Process just the `+` token if the next character is not `=`
			output = append(output, Token{Kind: PlusToken, Value: string(char), Line: line, Column: column})
		} else if char == '-' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '=' {
					// Process the `+=` token
					output = append(output, Token{Kind: MinusEqualsToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			output = append(output, Token{Kind: MinusToken, Value: string(char), Line: line, Column: column})
		} else if char == '*' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '=' {
					// Process the `+=` token
					output = append(output, Token{Kind: MultiplyEqualsToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			output = append(output, Token{Kind: MultiplyToken, Value: string(char), Line: line, Column: column})
		} else if char == '/' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '=' {
					// Process the `+=` token
					output = append(output, Token{Kind: DivideEqualsToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			output = append(output, Token{Kind: DivideToken, Value: string(char), Line: line, Column: column})
		} else if char == '%' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '=' {
					// Process the `+=` token
					output = append(output, Token{Kind: ModuloEqualsToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			output = append(output, Token{Kind: ModuloToken, Value: string(char), Line: line, Column: column})
		} else if char == '=' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '=' {
					// Process the `+=` token
					output = append(output, Token{Kind: IsAltToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			output = append(output, Token{Kind: AssignmentToken, Value: string(char), Line: line, Column: column})
		} else if char == '!' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '=' {
					// Process the `+=` token
					output = append(output, Token{Kind: NotEqualsAltToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			output = append(output, Token{Kind: NotToken, Value: string(char), Line: line, Column: column})
		} else if char == '<' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '=' {
					// Process the `+=` token
					output = append(output, Token{Kind: LessThanEqualsToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			output = append(output, Token{Kind: LessThanToken, Value: string(char), Line: line, Column: column})
		} else if char == '>' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '=' {
					// Process the `+=` token
					output = append(output, Token{Kind: GreaterThanEqualsToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			output = append(output, Token{Kind: GreaterThanToken, Value: string(char), Line: line, Column: column})
		} else if char == '&' {

			// Check if the next character is `&`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '&' {
					// Process the `&&` token
					output = append(output, Token{Kind: AndAltToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}

			output = append(output, Token{Kind: ReferenceToken, Value: string(char), Line: line, Column: column})
		} else if char == '?' {
			output = append(output, Token{Kind: QuestionMarkToken, Value: string(char), Line: line, Column: column})
		} else if char == '|' {

			// Check if the next character is `=`
			if i+1 < length {
				nextChar, nextSize := utf8.DecodeRuneInString(input[i+1:])

				if nextChar == '|' {
					// Process the `+=` token
					output = append(output, Token{Kind: OrAltToken, Value: string(char) + string(nextChar), Line: line, Column: column})
					// Move i forward by the size of both runes
					i += size + nextSize
					continue
				}
			}
		} else {

			// Number
			// Check if the next character is a digit (0-9)
			// then we need to read the digits until we encounter a non-digit character
			if unicode.IsDigit(char) {

				value := ""

				for j := i; j < length; j++ {
					char, _ = utf8.DecodeRuneInString(input[j:])
					if !unicode.IsDigit(char) {
						break
					}
					value += string(char)
				}

				output = append(output, Token{Kind: NumberToken, Value: value, Line: line, Column: column})

				// increment i by the number of bytes the rune that `value` occupies
				i += utf8.RuneCountInString(value)

				continue
			}

			// Identifier
			// Check if the current character is a letter, digit, or underscore
			// If it is, continue to the next character
			if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_' {

				value := ""

				for j := i; j < length; j++ {
					char, _ = utf8.DecodeRuneInString(input[j:])
					if !(unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_') {
						break
					}
					value += string(char)
				}

				// increment i by the number of bytes the rune that `value` occupies
				i += utf8.RuneCountInString(value)

				// Check if the identifier is a reserved word
				if kind, ok := ReservedWords[value]; ok {
					output = append(output, Token{Kind: kind, Value: value, Line: line, Column: column})
					continue
				}

				output = append(output, Token{Kind: IdentifierToken, Value: value, Line: line, Column: column})

				continue
			} else {
				panic(fmt.Sprintf("Unexpected character: %s (%d:%d)", string(char), line, column))
			}

		}
		// Move i forward by the number of bytes the rune occupies
		i += size
	}

	output = append(output, Token{Kind: EOFToken, Value: "EOF", Line: line, Column: column})

	return output
}
