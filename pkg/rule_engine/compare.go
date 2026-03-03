package rule_engine

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// EvaluateCondition parses and evaluates a condition string against attributes.
func EvaluateCondition(condition string, attributes map[string]interface{}) (bool, error) {
	parser := newParser(condition)
	expr, err := parser.parseExpression()
	if err != nil {
		return false, err
	}
	if parser.current().typ != tokenEOF {
		return false, fmt.Errorf("unexpected token: %s", parser.current().value)
	}
	return expr.evaluate(attributes)
}

type tokenType string

const (
	tokenEOF        tokenType = "EOF"
	tokenIdentifier tokenType = "IDENT"
	tokenString     tokenType = "STRING"
	tokenNumber     tokenType = "NUMBER"
	tokenBool       tokenType = "BOOL"
	tokenOperator   tokenType = "OP"
	tokenLParen     tokenType = "LPAREN"
	tokenRParen     tokenType = "RPAREN"
)

type token struct {
	typ   tokenType
	value string
}

type parser struct {
	tokens []token
	pos    int
}

func newParser(input string) *parser {
	lexer := newLexer(input)
	return &parser{tokens: lexer.tokenize(), pos: 0}
}

func (p *parser) current() token {
	if p.pos >= len(p.tokens) {
		return token{typ: tokenEOF, value: ""}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() token {
	current := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return current
}

func (p *parser) expect(tt tokenType, value string) (token, error) {
	current := p.current()
	if current.typ != tt {
		return token{}, fmt.Errorf("expected token %s but got %s", tt, current.typ)
	}
	if value != "" && strings.ToUpper(current.value) != value {
		return token{}, fmt.Errorf("expected %s but got %s", value, current.value)
	}
	p.advance()
	return current, nil
}

func (p *parser) parseExpression() (expression, error) {
	return p.parseOr()
}

func (p *parser) parseOr() (expression, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for strings.EqualFold(p.current().value, "OR") {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &logicalExpression{operator: "OR", left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (expression, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for strings.EqualFold(p.current().value, "AND") {
		p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &logicalExpression{operator: "AND", left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (expression, error) {
	if strings.EqualFold(p.current().value, "NOT") {
		p.advance()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &unaryExpression{operator: "NOT", expr: expr}, nil
	}
	return p.parseComparison()
}

func (p *parser) parseComparison() (expression, error) {
	if p.current().typ == tokenLParen {
		p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenRParen, ""); err != nil {
			return nil, err
		}
		return expr, nil
	}
	// Handle standalone boolean literals (true/false)
	if p.current().typ == tokenBool {
		boolVal, err := strconv.ParseBool(p.current().value)
		if err != nil {
			return nil, err
		}
		p.advance()
		return &literalBoolExpression{value: boolVal}, nil
	}
	left, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	operatorToken := p.current()
	if operatorToken.typ != tokenOperator {
		return nil, fmt.Errorf("expected operator but got %s", operatorToken.value)
	}
	p.advance()
	right, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	return &comparisonExpression{left: left, operator: operatorToken.value, right: right}, nil
}

func (p *parser) parseValue() (valueExpression, error) {
	current := p.current()
	switch current.typ {
	case tokenIdentifier:
		p.advance()
		return &identifierValue{name: current.value}, nil
	case tokenString:
		p.advance()
		return &literalValue{value: current.value, literalType: tokenString}, nil
	case tokenNumber:
		p.advance()
		return &literalValue{value: current.value, literalType: tokenNumber}, nil
	case tokenBool:
		p.advance()
		return &literalValue{value: current.value, literalType: tokenBool}, nil
	default:
		return nil, fmt.Errorf("unexpected value token: %s", current.value)
	}
}

type lexer struct {
	input string
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: input, pos: 0}
}

func (l *lexer) tokenize() []token {
	var tokens []token
	for {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}
		ch := l.input[l.pos]
		switch {
		case isLetter(ch):
			identifier := l.readIdentifier()
			upper := strings.ToUpper(identifier)
			switch upper {
			case "AND", "OR", "NOT":
				tokens = append(tokens, token{typ: tokenOperator, value: upper})
			case "TRUE", "FALSE":
				tokens = append(tokens, token{typ: tokenBool, value: strings.ToLower(identifier)})
			default:
				tokens = append(tokens, token{typ: tokenIdentifier, value: identifier})
			}
		case isDigit(ch):
			number := l.readNumber()
			tokens = append(tokens, token{typ: tokenNumber, value: number})
		case ch == '\'' || ch == '"':
			str := l.readString(ch)
			tokens = append(tokens, token{typ: tokenString, value: str})
		case ch == '(':
			l.pos++
			tokens = append(tokens, token{typ: tokenLParen, value: "("})
		case ch == ')':
			l.pos++
			tokens = append(tokens, token{typ: tokenRParen, value: ")"})
		default:
			operator := l.readOperator()
			tokens = append(tokens, token{typ: tokenOperator, value: operator})
		}
	}
	tokens = append(tokens, token{typ: tokenEOF, value: ""})
	return tokens
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		if l.input[l.pos] != ' ' && l.input[l.pos] != '\t' && l.input[l.pos] != '\n' && l.input[l.pos] != '\r' {
			return
		}
		l.pos++
	}
}

func (l *lexer) readIdentifier() string {
	start := l.pos
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if isLetter(ch) || isDigit(ch) || ch == '_' || ch == '.' {
			l.pos++
			continue
		}
		break
	}
	return l.input[start:l.pos]
}

func (l *lexer) readNumber() string {
	start := l.pos
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if isDigit(ch) || ch == '.' {
			l.pos++
			continue
		}
		break
	}
	return l.input[start:l.pos]
}

func (l *lexer) readString(quote byte) string {
	l.pos++
	start := l.pos
	for l.pos < len(l.input) {
		if l.input[l.pos] == quote {
			value := l.input[start:l.pos]
			l.pos++
			return value
		}
		l.pos++
	}
	return l.input[start:l.pos]
}

func (l *lexer) readOperator() string {
	operators := []string{"==", "!=", ">=", "<=", ">", "<", "~=", "="}
	for _, op := range operators {
		if strings.HasPrefix(l.input[l.pos:], op) {
			l.pos += len(op)
			return op
		}
	}
	l.pos++
	return string(l.input[l.pos-1])
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

type expression interface {
	evaluate(attributes map[string]interface{}) (bool, error)
}

type valueExpression interface {
	resolve(attributes map[string]interface{}) (interface{}, error)
}

type logicalExpression struct {
	operator string
	left     expression
	right    expression
}

func (expr *logicalExpression) evaluate(attributes map[string]interface{}) (bool, error) {
	left, err := expr.left.evaluate(attributes)
	if err != nil {
		return false, err
	}
	if strings.EqualFold(expr.operator, "AND") {
		if !left {
			return false, nil
		}
		right, err := expr.right.evaluate(attributes)
		if err != nil {
			return false, err
		}
		return left && right, nil
	}
	if strings.EqualFold(expr.operator, "OR") {
		if left {
			return true, nil
		}
		right, err := expr.right.evaluate(attributes)
		if err != nil {
			return false, err
		}
		return left || right, nil
	}
	return false, fmt.Errorf("unsupported logical operator: %s", expr.operator)
}

type unaryExpression struct {
	operator string
	expr     expression
}

func (expr *unaryExpression) evaluate(attributes map[string]interface{}) (bool, error) {
	value, err := expr.expr.evaluate(attributes)
	if err != nil {
		return false, err
	}
	if strings.EqualFold(expr.operator, "NOT") {
		return !value, nil
	}
	return false, fmt.Errorf("unsupported unary operator: %s", expr.operator)
}

type literalBoolExpression struct {
	value bool
}

func (expr *literalBoolExpression) evaluate(attributes map[string]interface{}) (bool, error) {
	return expr.value, nil
}

type comparisonExpression struct {
	left     valueExpression
	operator string
	right    valueExpression
}

func (expr *comparisonExpression) evaluate(attributes map[string]interface{}) (bool, error) {
	left, err := expr.left.resolve(attributes)
	if err != nil {
		return false, err
	}
	right, err := expr.right.resolve(attributes)
	if err != nil {
		return false, err
	}
	return compareValues(left, right, expr.operator)
}

type identifierValue struct {
	name string
}

func (value *identifierValue) resolve(attributes map[string]interface{}) (interface{}, error) {
	if strings.HasPrefix(value.name, "entity.attributes.") {
		key := strings.TrimPrefix(value.name, "entity.attributes.")
		return getAttributeValue(attributes, key)
	}
	return getAttributeValue(attributes, value.name)
}

type literalValue struct {
	value       string
	literalType tokenType
}

func (value *literalValue) resolve(attributes map[string]interface{}) (interface{}, error) {
	switch value.literalType {
	case tokenString:
		return value.value, nil
	case tokenNumber:
		parsed, err := strconv.ParseFloat(value.value, 64)
		if err != nil {
			return nil, err
		}
		return parsed, nil
	case tokenBool:
		parsed, err := strconv.ParseBool(value.value)
		if err != nil {
			return nil, err
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("unsupported literal type: %s", value.literalType)
	}
}

func getAttributeValue(attributes map[string]interface{}, key string) (interface{}, error) {
	if value, ok := attributes[key]; ok {
		return value, nil
	}
	return nil, fmt.Errorf("attribute not found: %s", key)
}

func compareValues(left interface{}, right interface{}, operator string) (bool, error) {
	switch operator {
	case "==", "!=":
		return compareEquality(left, right, operator)
	case ">", "<", ">=", "<=":
		return compareOrder(left, right, operator)
	case "~=" :
		return compareRegex(left, right)
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

func compareEquality(left interface{}, right interface{}, operator string) (bool, error) {
	leftVal, rightVal, err := normalizeComparable(left, right)
	if err != nil {
		return false, err
	}
	result := leftVal == rightVal
	if operator == "!=" {
		return !result, nil
	}
	return result, nil
}

func compareOrder(left interface{}, right interface{}, operator string) (bool, error) {
	leftFloat, rightFloat, ok := toFloat(left, right)
	if ok {
		return compareFloats(leftFloat, rightFloat, operator), nil
	}
	leftTime, rightTime, ok := toTime(left, right)
	if ok {
		return compareTimes(leftTime, rightTime, operator), nil
	}
	return false, errors.New("order comparison requires numeric or date values")
}

func compareRegex(left interface{}, right interface{}) (bool, error) {
	leftStr, ok := left.(string)
	if !ok {
		return false, errors.New("left value must be a string for regex comparison")
	}
	pattern, ok := right.(string)
	if !ok {
		return false, errors.New("right value must be a string for regex comparison")
	}
	matched, err := regexp.MatchString(pattern, leftStr)
	if err != nil {
		return false, err
	}
	return matched, nil
}

func normalizeComparable(left interface{}, right interface{}) (string, string, error) {
	if left == nil || right == nil {
		return "", "", errors.New("cannot compare nil values")
	}
	if leftBool, ok := left.(bool); ok {
		rightBool, ok := right.(bool)
		if !ok {
			return "", "", errors.New("type mismatch for boolean comparison")
		}
		return strconv.FormatBool(leftBool), strconv.FormatBool(rightBool), nil
	}
	if leftStr, ok := left.(string); ok {
		rightStr, ok := right.(string)
		if !ok {
			return "", "", errors.New("type mismatch for string comparison")
		}
		return leftStr, rightStr, nil
	}
	leftFloat, rightFloat, ok := toFloat(left, right)
	if ok {
		return strconv.FormatFloat(leftFloat, 'f', -1, 64), strconv.FormatFloat(rightFloat, 'f', -1, 64), nil
	}
	leftTime, rightTime, ok := toTime(left, right)
	if ok {
		return leftTime.Format(time.RFC3339), rightTime.Format(time.RFC3339), nil
	}
	return "", "", errors.New("unsupported types for equality comparison")
}

func toFloat(left interface{}, right interface{}) (float64, float64, bool) {
	leftFloat, ok := convertToFloat(left)
	if !ok {
		return 0, 0, false
	}
	rightFloat, ok := convertToFloat(right)
	if !ok {
		return 0, 0, false
	}
	return leftFloat, rightFloat, true
}

func convertToFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func toTime(left interface{}, right interface{}) (time.Time, time.Time, bool) {
	leftTime, ok := convertToTime(left)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	rightTime, ok := convertToTime(right)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	return leftTime, rightTime, true
}

func convertToTime(value interface{}) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case string:
		parsed, err := time.Parse("2006-01-02", v)
		if err != nil {
			return time.Time{}, false
		}
		return parsed, true
	default:
		return time.Time{}, false
	}
}

func compareFloats(left float64, right float64, operator string) bool {
	switch operator {
	case ">":
		return left > right
	case "<":
		return left < right
	case ">=":
		return left >= right
	case "<=":
		return left <= right
	default:
		return false
	}
}

func compareTimes(left time.Time, right time.Time, operator string) bool {
	switch operator {
	case ">":
		return left.After(right)
	case "<":
		return left.Before(right)
	case ">=":
		return left.After(right) || left.Equal(right)
	case "<=":
		return left.Before(right) || left.Equal(right)
	default:
		return false
	}
}
