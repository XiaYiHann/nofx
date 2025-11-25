package decision

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractDecisions(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedLen    int
		expectedAction string
		expectError    bool
	}{
		{
			name:           "Standard JSON block",
			input:          "Here is the decision:\n```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"open_long\", \"reasoning\": \"test\"}]\n```",
			expectedLen:    1,
			expectedAction: "open_long",
			expectError:    false,
		},
		{
			name:           "JSON without block",
			input:          "Some reasoning...\n[{\"symbol\": \"ETHUSDT\", \"action\": \"close_short\", \"reasoning\": \"test\"}]",
			expectedLen:    1,
			expectedAction: "close_short",
			expectError:    false,
		},
		{
			name:           "XML tags",
			input:          "<reasoning>Thinking...</reasoning>\n<decision>\n```json\n[{\"symbol\": \"SOLUSDT\", \"action\": \"hold\", \"reasoning\": \"wait\"}]\n```\n</decision>",
			expectedLen:    1,
			expectedAction: "hold",
			expectError:    false,
		},
		{
			name:           "Full-width characters fix",
			input:          "［｛＂symbol＂： ＂BTCUSDT＂， ＂action＂： ＂wait＂， ＂reasoning＂： ＂test＂｝］",
			expectedLen:    1,
			expectedAction: "wait",
			expectError:    false,
		},
		{
			name:           "Malformed JSON fallback",
			input:          "I cannot decide.",
			expectedLen:    1,
			expectedAction: "wait", // Should fallback to wait
			expectError:    false,
		},
		{
			name:           "Empty input fallback",
			input:          "",
			expectedLen:    1,
			expectedAction: "wait",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decisions, err := extractDecisions(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, decisions, tt.expectedLen)
				if len(decisions) > 0 {
					assert.Equal(t, tt.expectedAction, decisions[0].Action)
				}
			}
		})
	}
}

func TestParseFullDecisionResponse(t *testing.T) {
	input := `<reasoning>
Market is bullish.
MACD is positive.
</reasoning>

<decision>
` + "```json" + `
[
  {
    "symbol": "BTCUSDT",
    "action": "open_long",
    "leverage": 5,
    "position_size_usd": 1000,
    "stop_loss": 49000,
    "take_profit": 55000,
    "confidence": 80,
    "risk_usd": 100,
    "reasoning": "Bullish trend"
  }
]
` + "```" + `
</decision>`

	// Mock account equity and leverage for validation
	equity := 10000.0
	btcEthLev := 10
	altLev := 5
	minRR := 3.0

	fullDecision, err := parseFullDecisionResponse(input, equity, btcEthLev, altLev, minRR)
	assert.NoError(t, err)
	assert.NotNil(t, fullDecision)
	assert.Contains(t, fullDecision.CoTTrace, "Market is bullish")
	assert.Len(t, fullDecision.Decisions, 1)
	assert.Equal(t, "open_long", fullDecision.Decisions[0].Action)
}

func TestExtractCoTTrace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "XML reasoning tag",
			input:    "<reasoning>My thought process</reasoning><decision>...</decision>",
			expected: "My thought process",
		},
		{
			name:     "Before decision tag",
			input:    "My thought process\n<decision>...</decision>",
			expected: "My thought process",
		},
		{
			name:     "Before JSON array",
			input:    "My thought process\n[{\"action\": \"wait\"}]",
			expected: "My thought process",
		},
		{
			name:     "No tags",
			input:    "Just thoughts",
			expected: "Just thoughts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCoTTrace(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
