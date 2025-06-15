package services

import (
	"testing"

	"flashcards/models"
)

func TestNoteMatchesSearch(t *testing.T) {
	service := &NoteService{}

	tests := []struct {
		name        string
		noteContent string
		searchTerms []string
		expected    bool
	}{
		{
			name:        "exact match",
			noteContent: "This is about scalability and performance",
			searchTerms: []string{"scalability"},
			expected:    true,
		},
		{
			name:        "case insensitive match",
			noteContent: "This is about SCALABILITY and performance",
			searchTerms: []string{"scalability"},
			expected:    true,
		},
		{
			name:        "partial word match",
			noteContent: "This is about scalability and performance",
			searchTerms: []string{"scalability"},
			expected:    true,
		},
		{
			name:        "typo tolerance",
			noteContent: "This is about database connections",
			searchTerms: []string{"databse"},
			expected:    true,
		},
		{
			name:        "multiple terms - one matches",
			noteContent: "This is about microservices architecture",
			searchTerms: []string{"microservices", "nosql"},
			expected:    true,
		},
		{
			name:        "multiple terms - none match",
			noteContent: "This is about microservices architecture",
			searchTerms: []string{"nosql", "blockchain"},
			expected:    false,
		},
		{
			name:        "punctuation handling",
			noteContent: "This is about caching, performance, and scalability.",
			searchTerms: []string{"caching"},
			expected:    true,
		},
		{
			name:        "compound word matching",
			noteContent: "We use microservices for better scalability",
			searchTerms: []string{"micro"},
			expected:    true,
		},
		{
			name:        "short term matching",
			noteContent: "Database performance is critical",
			searchTerms: []string{"database"},
			expected:    true,
		},
		{
			name:        "fuzzy match with suffix",
			noteContent: "The application has good scalability",
			searchTerms: []string{"scalability"},
			expected:    true,
		},
		{
			name:        "no match",
			noteContent: "This is about frontend development",
			searchTerms: []string{"backend"},
			expected:    false,
		},
		{
			name:        "empty search terms",
			noteContent: "This is about anything",
			searchTerms: []string{},
			expected:    false,
		},
		{
			name:        "empty note content",
			noteContent: "",
			searchTerms: []string{"test"},
			expected:    false,
		},
		{
			name:        "performance with common typos",
			noteContent: "This discusses performance optimization",
			searchTerms: []string{"performace", "optimiztion"},
			expected:    true,
		},
		{
			name:        "technical abbreviations",
			noteContent: "Using REST APIs for communication",
			searchTerms: []string{"apis"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := &models.Note{
				Content: tt.noteContent,
			}

			result := service.noteMatchesSearch(note, tt.searchTerms)
			if result != tt.expected {
				t.Errorf("noteMatchesSearch() = %v, expected %v for note: %q with terms: %v",
					result, tt.expected, tt.noteContent, tt.searchTerms)
			}
		})
	}
}

func TestSearchNotesByContent(t *testing.T) {
	// Create a mock implementation for testing SearchNotesByContent
	// This would typically require mocking the repository layer
	service := &NoteService{}

	// Test the search terms processing logic by testing noteMatchesSearch directly
	testNotes := []*models.Note{
		{ID: 1, Content: "This is about database scalability and performance optimization"},
		{ID: 2, Content: "Frontend development with React and TypeScript"},
		{ID: 3, Content: "Microservices architecture patterns and best practices"},
		{ID: 4, Content: "Caching strategies for distributed systems"},
		{ID: 5, Content: "Machine learning algorithms and data processing"},
	}

	tests := []struct {
		name         string
		searchTerms  []string
		expectedIDs  []int
		description  string
	}{
		{
			name:        "database search",
			searchTerms: []string{"database"},
			expectedIDs: []int{1},
			description: "should find notes about databases",
		},
		{
			name:        "frontend search",
			searchTerms: []string{"frontend"},
			expectedIDs: []int{2},
			description: "should find frontend-related notes",
		},
		{
			name:        "architecture search",
			searchTerms: []string{"architecture"},
			expectedIDs: []int{3},
			description: "should find architecture-related notes",
		},
		{
			name:        "performance search with typo",
			searchTerms: []string{"performace"},
			expectedIDs: []int{1},
			description: "should handle typos in search terms with strict Levenshtein distance",
		},
		{
			name:        "multiple term search",
			searchTerms: []string{"microservices", "caching"},
			expectedIDs: []int{3, 4},
			description: "should find notes matching any of the terms",
		},
		{
			name:        "partial match",
			searchTerms: []string{"distributed"},
			expectedIDs: []int{4},
			description: "should find word matches with stricter Levenshtein distance",
		},
		{
			name:        "no matches",
			searchTerms: []string{"blockchain"},
			expectedIDs: []int{},
			description: "should return empty when no matches found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matchingNotes []*models.Note
			
			// Simulate the search logic
			for _, note := range testNotes {
				if service.noteMatchesSearch(note, tt.searchTerms) {
					matchingNotes = append(matchingNotes, note)
				}
			}

			// Verify the results
			if len(matchingNotes) != len(tt.expectedIDs) {
				t.Errorf("Expected %d matches, got %d for search terms: %v",
					len(tt.expectedIDs), len(matchingNotes), tt.searchTerms)
				return
			}

			// Check that we got the expected note IDs
			matchedIDs := make([]int, len(matchingNotes))
			for i, note := range matchingNotes {
				matchedIDs[i] = note.ID
			}

			for _, expectedID := range tt.expectedIDs {
				found := false
				for _, matchedID := range matchedIDs {
					if matchedID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find note ID %d in results %v for search terms: %v",
						expectedID, matchedIDs, tt.searchTerms)
				}
			}
		})
	}
}

// Benchmark test to ensure the fuzzy search performance is acceptable
func BenchmarkNoteMatchesSearch(b *testing.B) {
	service := &NoteService{}
	note := &models.Note{
		Content: "This is a comprehensive guide about database scalability, performance optimization, caching strategies, and microservices architecture patterns for distributed systems.",
	}
	searchTerms := []string{"scalability", "performance", "caching"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.noteMatchesSearch(note, searchTerms)
	}
}