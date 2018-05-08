
package token

import (
	"strconv"
)

type Token int

func (tkn Token) String() string {
	if 0 == tkn {
		return "UNKNOWN"
	}
	if tkn < Token(len(token2string)) {
		return token2string[tkn]
	}
	return "token(" + strconv.Itoa(int(tkn)) + ")"
}

func (tkn Token) precedence(in bool) int {

	switch tkn {
	case LOGICAL_OR:
		return 1

	case LOGICAL_AND:
		return 2

	case OR, OR_ASSIGN:
		return 3

	case EXCLUSIVE_OR:
		return 4

	case AND, AND_ASSIGN, AND_NOT, AND_NOT_ASSIGN:
		return 5

	case EQUAL,
		NOT_EQUAL,
		STRICT_EQUAL,
		STRICT_NOT_EQUAL:
		return 6

	case LESS, GREATER, LESS_OR_EQUAL, GREATER_OR_EQUAL, INSTANCEOF:
		return 7

	case IN:
		if in {
			return 7
		}
		return 0

	case SHIFT_LEFT, SHIFT_RIGHT, UNSIGNED_SHIFT_RIGHT:
		fallthrough
	case SHIFT_LEFT_ASSIGN, SHIFT_RIGHT_ASSIGN, UNSIGNED_SHIFT_RIGHT_ASSIGN:
		return 8

	case PLUS, MINUS, ADD_ASSIGN, SUBTRACT_ASSIGN:
		return 9

	case MULTIPLY, SLASH, REMAINDER, MULTIPLY_ASSIGN, QUOTIENT_ASSIGN, REMAINDER_ASSIGN:
		return 11
	}
	return 0
}

type _keyword struct {
	token         Token
	futureKeyword bool
	strict        bool
}

func IsKeyword(literal string) (Token, bool) {
	if keyword, exists := keywordTable[literal]; exists {
		if keyword.futureKeyword {
			return KEYWORD, keyword.strict
		}
		return keyword.token, false
	}
	return 0, false
}
