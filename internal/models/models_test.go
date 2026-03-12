package models

import "testing"

func TestTicket_DisplayKey(t *testing.T) {
	ticket := Ticket{
		ProjectPrefix: "PROJ",
		Number:        42,
	}
	if ticket.DisplayKey() != "PROJ-42" {
		t.Errorf("Expected PROJ-42, got %s", ticket.DisplayKey())
	}
}
