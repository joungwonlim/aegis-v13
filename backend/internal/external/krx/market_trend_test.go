package krx

import (
	"testing"
	"time"
)

func TestParseNetBuyVolume(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{"positive with comma", "+1,459,781", 1459781},
		{"negative with comma", "-1,240,182", -1240182},
		{"positive without sign", "1000000", 1000000},
		{"negative", "-500000", -500000},
		{"with spaces", " +1,234 ", 1234},
		{"zero", "0", 0},
		{"empty string", "", 0},
		{"invalid", "abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseNetBuyVolume(tt.input); got != tt.want {
				t.Errorf("parseNetBuyVolume(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMarketTrendDataStructure(t *testing.T) {
	now := time.Now()
	data := MarketTrendData{
		TradeDate:      now,
		ForeignNet:     100000.0,
		InstitutionNet: 50000.0,
		IndividualNet:  -150000.0,
	}

	// Verify fields
	if data.TradeDate.IsZero() {
		t.Error("TradeDate should not be zero")
	}
	if data.ForeignNet != 100000.0 {
		t.Errorf("ForeignNet = %v, want 100000.0", data.ForeignNet)
	}
	if data.InstitutionNet != 50000.0 {
		t.Errorf("InstitutionNet = %v, want 50000.0", data.InstitutionNet)
	}
	if data.IndividualNet != -150000.0 {
		t.Errorf("IndividualNet = %v, want -150000.0", data.IndividualNet)
	}

	// Verify balance (sum should be 0)
	sum := data.ForeignNet + data.InstitutionNet + data.IndividualNet
	if sum != 0 {
		t.Errorf("Net sum = %v, want 0 (balanced)", sum)
	}
}

func TestMarketTrendResponseStructure(t *testing.T) {
	resp := MarketTrendResponse{
		Bizdate:          "20240115",
		PersonalValue:    "-150000",
		ForeignValue:     "+100000",
		InstitutionValue: "+50000",
	}

	// Verify all fields are accessible
	if resp.Bizdate != "20240115" {
		t.Errorf("Bizdate = %q, want 20240115", resp.Bizdate)
	}
	if resp.PersonalValue != "-150000" {
		t.Errorf("PersonalValue = %q, want -150000", resp.PersonalValue)
	}
	if resp.ForeignValue != "+100000" {
		t.Errorf("ForeignValue = %q, want +100000", resp.ForeignValue)
	}
	if resp.InstitutionValue != "+50000" {
		t.Errorf("InstitutionValue = %q, want +50000", resp.InstitutionValue)
	}
}
