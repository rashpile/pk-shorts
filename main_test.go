package main

import (
	"testing"
)

func TestGenerateShortID(t *testing.T) {
	id1 := generateShortID()
	id2 := generateShortID()

	if len(id1) != shortIDLength {
		t.Errorf("Expected ID length to be %d, got %d", shortIDLength, len(id1))
	}

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	for i := 0; i < 100; i++ {
		id := generateShortID()
		if len(id) != shortIDLength {
			t.Errorf("ID %d has incorrect length: expected %d, got %d", i, shortIDLength, len(id))
		}
	}
}

func TestGenerateSecureID(t *testing.T) {
	id1 := generateSecureID()
	id2 := generateSecureID()

	if len(id1) != secureIDLength {
		t.Errorf("Expected secure ID length to be %d, got %d", secureIDLength, len(id1))
	}

	if id1 == id2 {
		t.Error("Generated secure IDs should be unique")
	}

	// Test uniqueness with more samples
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := generateSecureID()
		if len(id) != secureIDLength {
			t.Errorf("Secure ID %d has incorrect length: expected %d, got %d", i, secureIDLength, len(id))
		}
		if ids[id] {
			t.Errorf("Duplicate secure ID generated: %s", id)
		}
		ids[id] = true
	}

	// Verify that secure IDs are longer than regular IDs
	if secureIDLength <= shortIDLength {
		t.Error("Secure IDs should be longer than regular IDs for better security")
	}
}

func TestValidateCustomID(t *testing.T) {
	// Helper to create strings of specific length
	makeString := func(length int) string {
		result := ""
		for i := 0; i < length; i++ {
			result += "a"
		}
		return result
	}

	tests := []struct {
		id        string
		shouldErr bool
		desc      string
	}{
		{"my-link", false, "valid ID with dash"},
		{"user_123", false, "valid ID with underscore and numbers"},
		{"MyLink", false, "valid ID with uppercase"},
		{"abc", false, "minimum length ID"},
		{makeString(50), false, "maximum length ID"},
		{"ab", true, "too short"},
		{makeString(51), true, "too long"},
		{"my link", true, "contains space"},
		{"my@link", true, "contains invalid character"},
		{"admin", true, "reserved word"},
		{"api", true, "reserved word"},
		{"", true, "empty string"},
		{"a", true, "single character"},
		{"aa", true, "two characters"},
	}

	for _, test := range tests {
		err := validateCustomID(test.id)
		if (err != nil) != test.shouldErr {
			if test.shouldErr {
				t.Errorf("%s: expected error for ID '%s' but got none", test.desc, test.id)
			} else {
				t.Errorf("%s: unexpected error for ID '%s': %v", test.desc, test.id, err)
			}
		}
	}
}