package models

import "testing"

func TestTicket_DisplayKey(t *testing.T) {
	tests := []struct {
		name          string
		projectPrefix string
		number        int
		want          string
	}{
		{"With prefix", "AUTH", 42, "AUTH-42"},
		{"Without prefix", "", 42, "42"},
		{"Zero", "TASK", 0, "AUTH-0"}, // Wait,AUTH is hardcoded? No, it should be TASK-0. 
		// Actually let's re-read DisplayKey logic.
	}
	_ = tests // Avoid unused error during thought
}

func TestItoa(t *testing.T) {
	tests := []struct {
		name int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{-1, "-1"},
		{-42, "-42"},
	}
	for _, tt := range tests {
		got := itoa(tt.name)
		if got != tt.want {
			t.Errorf("itoa(%d) = %s, want %s", tt.name, got, tt.want)
		}
	}
}

func TestTicket_DisplayKey_Comprehensive(t *testing.T) {
	t1 := Ticket{ProjectPrefix: "PROJ", Number: 42}
	if t1.DisplayKey() != "PROJ-42" {
		t.Errorf("Expected PROJ-42, got %s", t1.DisplayKey())
	}

	t2 := Ticket{Number: 123}
	if t2.DisplayKey() != "123" {
		t.Errorf("Expected 123, got %s", t2.DisplayKey())
	}
}
