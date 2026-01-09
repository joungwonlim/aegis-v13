package contracts

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTargetPortfolio_TotalWeight(t *testing.T) {
	portfolio := &TargetPortfolio{
		Date: time.Now(),
		Positions: []TargetPosition{
			{Code: "005930", Weight: 0.30},
			{Code: "000660", Weight: 0.25},
			{Code: "035420", Weight: 0.20},
		},
		Cash: 0.25,
	}

	expected := 0.30 + 0.25 + 0.20
	if total := portfolio.TotalWeight(); total != expected {
		t.Errorf("TotalWeight() = %v, want %v", total, expected)
	}
}

func TestTargetPortfolio_Count(t *testing.T) {
	portfolio := &TargetPortfolio{
		Date: time.Now(),
		Positions: []TargetPosition{
			{Code: "005930", Weight: 0.30},
			{Code: "000660", Weight: 0.25},
		},
	}

	if count := portfolio.Count(); count != 2 {
		t.Errorf("Count() = %d, want 2", count)
	}
}

func TestTargetPortfolio_GetPosition(t *testing.T) {
	portfolio := &TargetPortfolio{
		Date: time.Now(),
		Positions: []TargetPosition{
			{Code: "005930", Name: "Samsung", Weight: 0.30},
			{Code: "000660", Name: "SK Hynix", Weight: 0.25},
		},
	}

	// Existing position
	pos, exists := portfolio.GetPosition("005930")
	if !exists {
		t.Error("Expected to find position for 005930")
	}
	if pos.Name != "Samsung" {
		t.Errorf("Got name %s, want Samsung", pos.Name)
	}

	// Non-existing position
	_, exists = portfolio.GetPosition("999999")
	if exists {
		t.Error("Expected not to find position for 999999")
	}
}

func TestAction_Constants(t *testing.T) {
	if ActionBuy != "BUY" {
		t.Errorf("ActionBuy = %s, want BUY", ActionBuy)
	}
	if ActionSell != "SELL" {
		t.Errorf("ActionSell = %s, want SELL", ActionSell)
	}
	if ActionHold != "HOLD" {
		t.Errorf("ActionHold = %s, want HOLD", ActionHold)
	}
}

func TestTargetPortfolio_JSON(t *testing.T) {
	original := &TargetPortfolio{
		Date: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		Positions: []TargetPosition{
			{
				Code:      "005930",
				Name:      "Samsung",
				Weight:    0.30,
				TargetQty: 100,
				Action:    ActionBuy,
				Reason:    "Strong momentum",
			},
		},
		Cash: 0.25,
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded TargetPortfolio
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if decoded.Cash != original.Cash {
		t.Errorf("Cash mismatch: got %f, want %f", decoded.Cash, original.Cash)
	}
	if len(decoded.Positions) != len(original.Positions) {
		t.Errorf("Positions count mismatch: got %d, want %d", len(decoded.Positions), len(original.Positions))
	}
	if decoded.Positions[0].Code != original.Positions[0].Code {
		t.Errorf("Position code mismatch: got %s, want %s", decoded.Positions[0].Code, original.Positions[0].Code)
	}
}
