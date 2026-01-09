package naver

import (
	"testing"
	"time"
)

func TestParseInvestorHTML(t *testing.T) {
	// Sample HTML from Naver Finance investor page
	sampleHTML := `
		<html>
		<body>
		<table class="type2">
			<tr><th>Header</th></tr>
		</table>
		<table class="type2">
			<tr>
				<td>2024.01.15</td>
				<td>72,500</td>
				<td>+500</td>
				<td>+0.69%</td>
				<td>1,000,000</td>
				<td>+50,000</td>
				<td>+30,000</td>
			</tr>
			<tr>
				<td>2024.01.16</td>
				<td>73,000</td>
				<td>+500</td>
				<td>+0.69%</td>
				<td>1,200,000</td>
				<td>+60,000</td>
				<td>+40,000</td>
			</tr>
			<tr>
				<td>invalid date</td>
				<td>73,000</td>
			</tr>
		</table>
		</body>
		</html>
	`

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	c := &Client{}
	trades, lastDate, hasMore := c.parseInvestorHTML(sampleHTML, "005930", from, to)

	// Should parse 2 valid rows
	if len(trades) != 2 {
		t.Errorf("parseInvestorHTML() got %d trades, want 2", len(trades))
	}

	// Verify first trade
	if len(trades) > 0 {
		trade := trades[0]
		if trade.StockCode != "005930" {
			t.Errorf("StockCode = %s, want 005930", trade.StockCode)
		}
		expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		if !trade.TradeDate.Equal(expectedDate) {
			t.Errorf("TradeDate = %v, want %v", trade.TradeDate, expectedDate)
		}
		if trade.InstitutionNet != 50000 {
			t.Errorf("InstitutionNet = %d, want 50000", trade.InstitutionNet)
		}
		if trade.ForeignNet != 30000 {
			t.Errorf("ForeignNet = %d, want 30000", trade.ForeignNet)
		}
		// Individual = -(Foreign + Institution)
		expectedIndividual := int64(-(30000 + 50000))
		if trade.IndividualNet != expectedIndividual {
			t.Errorf("IndividualNet = %d, want %d", trade.IndividualNet, expectedIndividual)
		}
	}

	// Verify last date
	if lastDate.IsZero() {
		t.Error("parseInvestorHTML() lastDate is zero")
	}

	// hasMore should be false (no pagination links in sample)
	if hasMore {
		t.Error("parseInvestorHTML() hasMore = true, want false")
	}
}

func TestParseInvestorHTMLNoTables(t *testing.T) {
	html := "<html><body></body></html>"
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	c := &Client{}
	trades, lastDate, hasMore := c.parseInvestorHTML(html, "005930", from, to)

	if len(trades) != 0 {
		t.Errorf("parseInvestorHTML() got %d trades, want 0", len(trades))
	}
	if !lastDate.IsZero() {
		t.Error("parseInvestorHTML() lastDate should be zero")
	}
	if hasMore {
		t.Error("parseInvestorHTML() hasMore = true, want false")
	}
}

func TestParseInvestorHTMLDateFilter(t *testing.T) {
	html := `
		<html>
		<body>
		<table class="type2"></table>
		<table class="type2">
			<tr>
				<td>2024.01.15</td>
				<td>72,500</td>
				<td>+500</td>
				<td>+0.69%</td>
				<td>1,000,000</td>
				<td>+50,000</td>
				<td>+30,000</td>
			</tr>
			<tr>
				<td>2024.02.15</td>
				<td>73,000</td>
				<td>+500</td>
				<td>+0.69%</td>
				<td>1,200,000</td>
				<td>+60,000</td>
				<td>+40,000</td>
			</tr>
		</table>
		</body>
		</html>
	`

	// Filter: only January 2024
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	c := &Client{}
	trades, _, _ := c.parseInvestorHTML(html, "005930", from, to)

	// Should only get the January date
	if len(trades) != 1 {
		t.Errorf("parseInvestorHTML() with date filter got %d trades, want 1", len(trades))
	}

	if len(trades) > 0 {
		expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		if !trades[0].TradeDate.Equal(expectedDate) {
			t.Errorf("Filtered trade date = %v, want %v", trades[0].TradeDate, expectedDate)
		}
	}
}

func TestInvestorFlowDataStructure(t *testing.T) {
	now := time.Now()
	flow := InvestorFlowData{
		StockCode:      "005930",
		TradeDate:      now,
		ForeignNet:     100000,
		InstitutionNet: 50000,
		IndividualNet:  -150000,
		FinancialNet:   0,
		InsuranceNet:   0,
		TrustNet:       0,
		PensionNet:     0,
	}

	// Verify all fields are accessible
	if flow.StockCode != "005930" {
		t.Errorf("StockCode = %s, want 005930", flow.StockCode)
	}
	if flow.ForeignNet != 100000 {
		t.Errorf("ForeignNet = %d, want 100000", flow.ForeignNet)
	}
	if flow.InstitutionNet != 50000 {
		t.Errorf("InstitutionNet = %d, want 50000", flow.InstitutionNet)
	}
	if flow.IndividualNet != -150000 {
		t.Errorf("IndividualNet = %d, want -150000", flow.IndividualNet)
	}

	// Verify balance: Foreign + Institution + Individual should equal 0
	sum := flow.ForeignNet + flow.InstitutionNet + flow.IndividualNet
	if sum != 0 {
		t.Errorf("Net sum = %d, want 0 (balanced)", sum)
	}
}
