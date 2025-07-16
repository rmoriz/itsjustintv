package twitch

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckTagFilter(t *testing.T) {
	enricher := NewEnricher(nil, nil, nil)

	tests := []struct {
		name        string
		twitchTags  []string
		tagFilter   []string
		expected    bool
		description string
	}{
		{
			name:        "No filter - allow all",
			twitchTags:  []string{"English", "Gaming"},
			tagFilter:   []string{},
			expected:    true,
			description: "When no tag_filter is configured, should allow all streams",
		},
		{
			name:        "Exact match - case sensitive",
			twitchTags:  []string{"Science & Technology", "Software Development"},
			tagFilter:   []string{"Science & Technology"},
			expected:    true,
			description: "Exact case-sensitive match should pass",
		},
		{
			name:        "Case insensitive match",
			twitchTags:  []string{"Science & Technology", "Software Development"},
			tagFilter:   []string{"science & technology"},
			expected:    true,
			description: "Case-insensitive exact match should pass",
		},
		{
			name:        "No match - block",
			twitchTags:  []string{"English", "Gaming"},
			tagFilter:   []string{"Science & Technology"},
			expected:    false,
			description: "When no tags match filter, should block",
		},
		{
			name:        "Partial match - block",
			twitchTags:  []string{"Science"},
			tagFilter:   []string{"Science & Technology"},
			expected:    false,
			description: "Partial match should not pass (exact match required)",
		},
		{
			name:        "Multiple filters - one match",
			twitchTags:  []string{"Gaming"},
			tagFilter:   []string{"Science & Technology", "Gaming", "Education"},
			expected:    true,
			description: "Should pass if at least one filter tag matches",
		},
		{
			name:        "Empty tags - no filter",
			twitchTags:  []string{},
			tagFilter:   []string{"Science & Technology"},
			expected:    false,
			description: "Empty tags with filter should block",
		},
		{
			name:        "Multiple tags - one matches filter",
			twitchTags:  []string{"English", "Science & Technology", "Gaming"},
			tagFilter:   []string{"Gaming"},
			expected:    true,
			description: "Should pass if any Twitch tag matches any filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enricher.checkTagFilter(tt.twitchTags, tt.tagFilter)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestEnrichPayloadWithTagFiltering(t *testing.T) {
	// This would be an integration test with mock Twitch API
	// For now, we'll test the tag filtering logic separately
	// The actual integration test would be in server_test.go
}

// Test case-insensitive matching
func TestCaseInsensitiveTagMatching(t *testing.T) {
	enricher := NewEnricher(nil, nil, nil)

	twitchTags := []string{"Science & Technology", "English"}
	tagFilter := []string{"SCIENCE & TECHNOLOGY", "english"}

	result := enricher.checkTagFilter(twitchTags, tagFilter)
	assert.True(t, result, "Should match case-insensitive")
}

// Test exact vs partial matching
func TestExactTagMatching(t *testing.T) {
	enricher := NewEnricher(nil, nil, nil)

	// These should NOT match (partial vs exact)
	twitchTags := []string{"Science"}
	tagFilter := []string{"Science & Technology"}

	result := enricher.checkTagFilter(twitchTags, tagFilter)
	assert.False(t, result, "Should not match partial strings")

	// These should match (exact)
	twitchTags = []string{"Science & Technology"}
	tagFilter = []string{"Science & Technology"}

	result = enricher.checkTagFilter(twitchTags, tagFilter)
	assert.True(t, result, "Should match exact strings")
}
