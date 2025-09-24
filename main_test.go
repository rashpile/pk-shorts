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