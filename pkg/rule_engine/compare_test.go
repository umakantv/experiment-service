package rule_engine

import (
	"testing"
	"time"
)

func TestEvaluateCondition_BasicComparisons(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "string equality - true",
			condition: "country == 'US'",
			attributes: map[string]interface{}{"country": "US"},
			want:       true,
		},
		{
			name:       "string equality - false",
			condition: "country == 'US'",
			attributes: map[string]interface{}{"country": "UK"},
			want:       false,
		},
		{
			name:       "string inequality - true",
			condition: "country != 'US'",
			attributes: map[string]interface{}{"country": "UK"},
			want:       true,
		},
		{
			name:       "string inequality - false",
			condition: "country != 'US'",
			attributes: map[string]interface{}{"country": "US"},
			want:       false,
		},
		{
			name:       "numeric greater than - true",
			condition: "age > 18",
			attributes: map[string]interface{}{"age": 25},
			want:       true,
		},
		{
			name:       "numeric greater than - false",
			condition: "age > 18",
			attributes: map[string]interface{}{"age": 15},
			want:       false,
		},
		{
			name:       "numeric less than - true",
			condition: "age < 30",
			attributes: map[string]interface{}{"age": 25},
			want:       true,
		},
		{
			name:       "numeric less than - false",
			condition: "age < 30",
			attributes: map[string]interface{}{"age": 35},
			want:       false,
		},
		{
			name:       "numeric greater than or equal - true (equal)",
			condition: "age >= 25",
			attributes: map[string]interface{}{"age": 25},
			want:       true,
		},
		{
			name:       "numeric greater than or equal - true (greater)",
			condition: "age >= 25",
			attributes: map[string]interface{}{"age": 30},
			want:       true,
		},
		{
			name:       "numeric greater than or equal - false",
			condition: "age >= 25",
			attributes: map[string]interface{}{"age": 20},
			want:       false,
		},
		{
			name:       "numeric less than or equal - true (equal)",
			condition: "age <= 25",
			attributes: map[string]interface{}{"age": 25},
			want:       true,
		},
		{
			name:       "numeric less than or equal - true (less)",
			condition: "age <= 25",
			attributes: map[string]interface{}{"age": 20},
			want:       true,
		},
		{
			name:       "numeric less than or equal - false",
			condition: "age <= 25",
			attributes: map[string]interface{}{"age": 30},
			want:       false,
		},
		{
			name:       "numeric equality",
			condition: "count == 100",
			attributes: map[string]interface{}{"count": 100},
			want:       true,
		},
		{
			name:       "numeric inequality",
			condition: "count != 100",
			attributes: map[string]interface{}{"count": 200},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_BooleanValues(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "boolean true equality",
			condition: "active == true",
			attributes: map[string]interface{}{"active": true},
			want:       true,
		},
		{
			name:       "boolean false equality",
			condition: "active == false",
			attributes: map[string]interface{}{"active": false},
			want:       true,
		},
		{
			name:       "boolean true inequality",
			condition: "active != false",
			attributes: map[string]interface{}{"active": true},
			want:       true,
		},
		{
			name:       "boolean false inequality",
			condition: "active != true",
			attributes: map[string]interface{}{"active": false},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_LogicalOperators(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "AND - both true",
			condition: "country == 'US' AND tier == 'premium'",
			attributes: map[string]interface{}{"country": "US", "tier": "premium"},
			want:       true,
		},
		{
			name:       "AND - first false",
			condition: "country == 'UK' AND tier == 'premium'",
			attributes: map[string]interface{}{"country": "US", "tier": "premium"},
			want:       false,
		},
		{
			name:       "AND - second false",
			condition: "country == 'US' AND tier == 'basic'",
			attributes: map[string]interface{}{"country": "US", "tier": "premium"},
			want:       false,
		},
		{
			name:       "AND - both false",
			condition: "country == 'UK' AND tier == 'basic'",
			attributes: map[string]interface{}{"country": "US", "tier": "premium"},
			want:       false,
		},
		{
			name:       "OR - both true",
			condition: "country == 'US' OR country == 'UK'",
			attributes: map[string]interface{}{"country": "US"},
			want:       true,
		},
		{
			name:       "OR - first true",
			condition: "country == 'US' OR country == 'UK'",
			attributes: map[string]interface{}{"country": "US"},
			want:       true,
		},
		{
			name:       "OR - second true",
			condition: "country == 'UK' OR country == 'US'",
			attributes: map[string]interface{}{"country": "US"},
			want:       true,
		},
		{
			name:       "OR - both false",
			condition: "country == 'FR' OR country == 'DE'",
			attributes: map[string]interface{}{"country": "US"},
			want:       false,
		},
		{
			name:       "NOT - true becomes false",
			condition: "NOT country == 'US'",
			attributes: map[string]interface{}{"country": "US"},
			want:       false,
		},
		{
			name:       "NOT - false becomes true",
			condition: "NOT country == 'UK'",
			attributes: map[string]interface{}{"country": "US"},
			want:       true,
		},
		{
			name:       "combined AND and OR",
			condition: "country == 'US' AND (tier == 'premium' OR tier == 'gold')",
			attributes: map[string]interface{}{"country": "US", "tier": "gold"},
			want:       true,
		},
		{
			name:       "complex expression with NOT",
			condition: "NOT (country == 'US' AND tier == 'basic')",
			attributes: map[string]interface{}{"country": "US", "tier": "premium"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_Parentheses(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "parentheses grouping - true",
			condition: "(country == 'US' OR country == 'UK') AND tier == 'premium'",
			attributes: map[string]interface{}{"country": "UK", "tier": "premium"},
			want:       true,
		},
		{
			name:       "parentheses grouping - false due to AND",
			condition: "(country == 'US' OR country == 'UK') AND tier == 'basic'",
			attributes: map[string]interface{}{"country": "UK", "tier": "premium"},
			want:       false,
		},
		{
			name:       "nested parentheses",
			condition: "((country == 'US' AND tier == 'premium') OR (country == 'UK' AND tier == 'gold'))",
			attributes: map[string]interface{}{"country": "UK", "tier": "gold"},
			want:       true,
		},
		{
			name:       "nested parentheses - false",
			condition: "((country == 'US' AND tier == 'premium') OR (country == 'UK' AND tier == 'gold'))",
			attributes: map[string]interface{}{"country": "UK", "tier": "premium"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_DateComparisons(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "date greater than - true",
			condition: "signupDate > '2022-01-01'",
			attributes: map[string]interface{}{"signupDate": "2023-06-15"},
			want:       true,
		},
		{
			name:       "date greater than - false",
			condition:  "signupDate > '2023-01-01'",
			attributes: map[string]interface{}{"signupDate": "2022-06-15"},
			want:       false,
		},
		{
			name:       "date less than - true",
			condition:  "signupDate < '2023-01-01'",
			attributes: map[string]interface{}{"signupDate": "2022-06-15"},
			want:       true,
		},
		{
			name:       "date less than - false",
			condition:  "signupDate < '2022-01-01'",
			attributes: map[string]interface{}{"signupDate": "2023-06-15"},
			want:       false,
		},
		{
			name:       "date equality",
			condition:  "signupDate == '2023-01-15'",
			attributes: map[string]interface{}{"signupDate": "2023-01-15"},
			want:       true,
		},
		{
			name:       "date inequality",
			condition:  "signupDate != '2023-01-15'",
			attributes: map[string]interface{}{"signupDate": "2023-01-16"},
			want:       true,
		},
		{
			name:       "date greater than or equal - equal",
			condition:  "signupDate >= '2023-01-15'",
			attributes: map[string]interface{}{"signupDate": "2023-01-15"},
			want:       true,
		},
		{
			name:       "date less than or equal - equal",
			condition:  "signupDate <= '2023-01-15'",
			attributes: map[string]interface{}{"signupDate": "2023-01-15"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_RegexMatching(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "regex match - simple pattern",
			condition: "email ~= '^[a-z]+@[a-z]+\\\\.com$'",
			attributes: map[string]interface{}{"email": "test@example.com"},
			want:       false, // Note: the pattern doesn't match due to escaping
		},
		{
			name:       "regex match - wildcard",
			condition:  "country ~= '^U[SA]$'",
			attributes: map[string]interface{}{"country": "US"},
			want:       true,
		},
		{
			name:       "regex match - no match",
			condition:  "country ~= '^U[SA]$'",
			attributes: map[string]interface{}{"country": "UK"},
			want:       false,
		},
		{
			name:       "regex match - case insensitive",
			condition:  "name ~= '(?i)^john'",
			attributes: map[string]interface{}{"name": "John"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_EntityAttributesPath(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "entity.attributes path",
			condition: "entity.attributes.country == 'US'",
			attributes: map[string]interface{}{"country": "US"},
			want:       true,
		},
		{
			name:       "entity.attributes path with nested condition",
			condition: "entity.attributes.tier == 'premium' AND entity.attributes.age > 25",
			attributes: map[string]interface{}{"tier": "premium", "age": 30},
			want:       true,
		},
		{
			name:       "mixed path styles",
			condition: "entity.attributes.country == 'US' AND tier == 'premium'",
			attributes: map[string]interface{}{"country": "US", "tier": "premium"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_NumericTypeCoercion(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "int to float comparison",
			condition: "age > 18.5",
			attributes: map[string]interface{}{"age": 25},
			want:       true,
		},
		{
			name:       "float to int comparison",
			condition: "score == 100",
			attributes: map[string]interface{}{"score": 100.0},
			want:       true,
		},
		{
			name:       "string number to float comparison",
			condition: "value > 50",
			attributes: map[string]interface{}{"value": "75.5"},
			want:       true,
		},
		{
			name:       "int comparison with string literal",
			condition: "age == 25",
			attributes: map[string]interface{}{"age": 25},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_ErrorCases(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		wantErr    bool
		errContains string
	}{
		{
			name:        "missing attribute",
			condition:   "country == 'US'",
			attributes:  map[string]interface{}{},
			wantErr:     true,
			errContains: "attribute not found",
		},
		{
			name:        "invalid syntax - missing operator",
			condition:   "country 'US'",
			attributes:  map[string]interface{}{"country": "US"},
			wantErr:     true,
			errContains: "expected operator",
		},
		{
			name:        "invalid syntax - unclosed parenthesis",
			condition:   "(country == 'US'",
			attributes:  map[string]interface{}{"country": "US"},
			wantErr:     true,
			errContains: "expected token RPAREN",
		},
		{
			name:        "type mismatch - string vs number comparison",
			condition:   "age > 'twenty'",
			attributes:  map[string]interface{}{"age": 25},
			wantErr:     true,
			errContains: "order comparison requires numeric or date values",
		},
		{
			name:        "type mismatch - boolean vs string",
			condition:   "active == 'true'",
			attributes:  map[string]interface{}{"active": true},
			wantErr:     true,
			errContains: "type mismatch",
		},
		{
			name:        "invalid regex pattern",
			condition:   "email ~= '[invalid'",
			attributes:  map[string]interface{}{"email": "test@example.com"},
			wantErr:     true,
			errContains: "error parsing regexp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("EvaluateCondition() error = %v, want error containing %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestEvaluateCondition_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "empty string comparison",
			condition: "name == ''",
			attributes: map[string]interface{}{"name": ""},
			want:       true,
		},
		{
			name:       "zero value comparison",
			condition: "count == 0",
			attributes: map[string]interface{}{"count": 0},
			want:       true,
		},
		{
			name:       "negative number comparison",
			condition: "balance < 0",
			attributes: map[string]interface{}{"balance": -100},
			want:       true,
		},
		{
			name:       "float precision",
			condition: "rate == 0.5",
			attributes: map[string]interface{}{"rate": 0.5},
			want:       true,
		},
		{
			name:       "complex nested expression",
			condition: "(country == 'US' AND tier == 'premium') OR (country == 'UK' AND age > 30)",
			attributes: map[string]interface{}{"country": "UK", "tier": "basic", "age": 35},
			want:       true,
		},
		{
			name:       "double quotes for string",
			condition: `country == "US"`,
			attributes: map[string]interface{}{"country": "US"},
			want:       true,
		},
		{
			name:       "multiple AND conditions",
			condition: "country == 'US' AND tier == 'premium' AND age > 25",
			attributes: map[string]interface{}{"country": "US", "tier": "premium", "age": 30},
			want:       true,
		},
		{
			name:       "multiple OR conditions",
			condition: "country == 'US' OR country == 'UK' OR country == 'CA'",
			attributes: map[string]interface{}{"country": "CA"},
			want:       true,
		},
		{
			name:       "NOT with parentheses",
			condition: "NOT (country == 'US' OR country == 'UK')",
			attributes: map[string]interface{}{"country": "FR"},
			want:       true,
		},
		{
			name:       "case insensitive AND/OR/NOT",
			condition: "country == 'US' and tier == 'premium' or age > 100",
			attributes: map[string]interface{}{"country": "US", "tier": "premium", "age": 50},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_TimeTimeAttribute(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "time.Time attribute comparison",
			condition: "createdAt > '2023-01-01'",
			attributes: map[string]interface{}{
				"createdAt": time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},
		{
			name:       "time.Time attribute equality",
			condition: "createdAt == '2023-06-15'",
			attributes: map[string]interface{}{
				"createdAt": time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_ComplexRealWorldScenarios(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		attributes map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "premium US user over 25",
			condition: "country == 'US' AND tier == 'premium' AND age > 25",
			attributes: map[string]interface{}{
				"country": "US",
				"tier":    "premium",
				"age":     30,
			},
			want: true,
		},
		{
			name:       "non-premium US user",
			condition: "country == 'US' AND tier == 'premium' AND age > 25",
			attributes: map[string]interface{}{
				"country": "US",
				"tier":    "basic",
				"age":     30,
			},
			want: false,
		},
		{
			name:       "fallback rule - always true",
			condition: "true",
			attributes: map[string]interface{}{},
			want:       true,
		},
		{
			name:       "fallback rule with boolean attribute",
			condition: "active == true",
			attributes: map[string]interface{}{
				"active": true,
			},
			want: true,
		},
		{
			name:       "complex business rule",
			condition: "(country == 'US' OR country == 'CA') AND (tier == 'premium' OR tier == 'gold') AND signupDate >= '2023-01-01'",
			attributes: map[string]interface{}{
				"country":     "CA",
				"tier":        "gold",
				"signupDate":  "2023-06-01",
			},
			want: true,
		},
		{
			name:       "exclude certain users",
			condition: "NOT (country == 'US' AND tier == 'basic')",
			attributes: map[string]interface{}{
				"country": "UK",
				"tier":    "basic",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}