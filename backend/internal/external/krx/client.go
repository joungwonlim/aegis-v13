package krx

import (
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Client handles communication with KRX data via Naver API
// ⭐ SSOT: KRX 시장 데이터 호출은 이 클라이언트에서만
type Client struct {
	httpClient *httputil.Client
	logger     *logger.Logger
	baseURL    string
}

// NewClient creates a new KRX client (via Naver API)
func NewClient(httpClient *httputil.Client, log *logger.Logger) *Client {
	return &Client{
		httpClient: httpClient,
		logger:     log,
		baseURL:    "https://m.stock.naver.com",
	}
}

// MarketTrendResponse represents market trend API response from Naver
type MarketTrendResponse struct {
	Bizdate            string `json:"bizdate"`            // Trade date (YYYYMMDD)
	PersonalValue      string `json:"personalValue"`      // Personal investor net trading value
	ForeignValue       string `json:"foreignValue"`       // Foreigner net trading value
	InstitutionValue string `json:"institutionalValue"` // Institutional net trading value
}

// MarketTrendData represents parsed market trend data
type MarketTrendData struct {
	TradeDate      time.Time
	ForeignNet     float64 // 외국인 순매수
	InstitutionNet float64 // 기관 순매수
	IndividualNet  float64 // 개인 순매수
}
