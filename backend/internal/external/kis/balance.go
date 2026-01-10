package kis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// TR IDs for balance queries
const (
	// 실전
	TRIDBalanceReal = "TTTC8434R"
	// 모의
	TRIDBalanceVirtual = "VTTC8434R"
)

// GetBalance returns account balance and positions
func (c *Client) GetBalance(ctx context.Context) (*Balance, []Position, error) {
	path := "/uapi/domestic-stock/v1/trading/inquire-balance"

	trID := TRIDBalanceReal
	if c.cfg.IsVirtual {
		trID = TRIDBalanceVirtual
	}

	// Account number format: first 8 digits + last 2 digits
	accountNo := c.cfg.AccountNo
	if len(accountNo) < 10 {
		return nil, nil, fmt.Errorf("invalid account number format")
	}
	cano := accountNo[:8]
	acntPrdtCd := accountNo[8:10]

	params := fmt.Sprintf("?CANO=%s&ACNT_PRDT_CD=%s&AFHR_FLPR_YN=N&OFL_YN=&INQR_DVSN=02&UNPR_DVSN=01&FUND_STTL_ICLD_YN=N&FNCG_AMT_AUTO_RDPT_YN=N&PRCS_DVSN=00&CTX_AREA_FK100=&CTX_AREA_NK100=",
		cano, acntPrdtCd)

	resp, err := c.request(ctx, http.MethodGet, path+params, trID, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("balance request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("balance API error status %d: %s", resp.StatusCode, string(body))
	}

	var result balanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("decode balance response: %w", err)
	}

	if result.RtCd != "0" {
		return nil, nil, fmt.Errorf("balance API error: %s - %s", result.MsgCd, result.Msg1)
	}

	// Parse balance
	balance := &Balance{}
	if len(result.Output2) > 0 {
		out := result.Output2[0]
		balance.TotalDeposit = parseIntSafe(out.DncaTotAmt)
		balance.AvailableCash = parseIntSafe(out.PrvsRcdlExccAmt)
		balance.TotalPurchase = parseIntSafe(out.PchsAmtSmtlAmt)
		balance.TotalEvaluation = parseIntSafe(out.EvluAmtSmtlAmt)
		balance.TotalProfitLoss = parseIntSafe(out.EvluPflsSmtlAmt)
		balance.TotalAsset = parseIntSafe(out.TotEvluAmt)

		if balance.TotalPurchase > 0 {
			balance.ProfitLossRate = float64(balance.TotalProfitLoss) / float64(balance.TotalPurchase) * 100
		}
	}

	// Parse positions
	positions := make([]Position, 0, len(result.Output1))
	for _, out := range result.Output1 {
		qty := parseIntSafe(out.HldgQty)
		if qty == 0 {
			continue // Skip zero quantity positions
		}

		pos := Position{
			StockCode:         out.Pdno,
			StockName:         out.PrdtName,
			Quantity:          qty,
			AvailableQuantity: parseIntSafe(out.OrdPsblQty),
			AvgBuyPrice:       parseIntSafe(out.PchsAvgPric),
			CurrentPrice:      parseIntSafe(out.Prpr),
			EvalAmount:        parseIntSafe(out.EvluAmt),
			PurchaseAmount:    parseIntSafe(out.PchsAmt),
			ProfitLoss:        parseIntSafe(out.EvluPflsAmt),
			ProfitLossRate:    parseFloatSafe(out.EvluPflsRt),
		}
		positions = append(positions, pos)
	}

	c.logger.WithFields(map[string]interface{}{
		"total_asset":     balance.TotalAsset,
		"positions_count": len(positions),
	}).Debug("Balance fetched")

	return balance, positions, nil
}

// GetPositions returns only positions (convenience method)
func (c *Client) GetPositions(ctx context.Context) ([]Position, error) {
	_, positions, err := c.GetBalance(ctx)
	return positions, err
}

// Helper functions
func parseIntSafe(s string) int64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func parseFloatSafe(s string) float64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
