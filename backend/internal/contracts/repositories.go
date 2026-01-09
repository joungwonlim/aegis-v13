package contracts

import (
	"context"
	"time"
)

// ⭐ SSOT: Repository 인터페이스 정의는 여기서만

// PriceRepository manages stock price data
type PriceRepository interface {
	GetByCodeAndDate(ctx context.Context, code string, date time.Time) (*Price, error)
	GetByCodeAndDateRange(ctx context.Context, code string, from, to time.Time) ([]*Price, error)
	GetLatestByCode(ctx context.Context, code string) (*Price, error)
	Save(ctx context.Context, price *Price) error
	SaveBatch(ctx context.Context, prices []*Price) error
}

// Price represents a stock price record
type Price struct {
	Code   string
	Date   time.Time
	Open   int64
	High   int64
	Low    int64
	Close  int64
	Volume int64
}

// InvestorFlowRepository manages investor flow (수급) data
type InvestorFlowRepository interface {
	GetByCodeAndDate(ctx context.Context, code string, date time.Time) (*InvestorFlow, error)
	GetByCodeAndDateRange(ctx context.Context, code string, from, to time.Time) ([]*InvestorFlow, error)
	Save(ctx context.Context, flow *InvestorFlow) error
	SaveBatch(ctx context.Context, flows []*InvestorFlow) error
}

// InvestorFlow represents investor buying/selling data
type InvestorFlow struct {
	Code          string
	Date          time.Time
	ForeignNet    int64 // 외국인 순매수
	InstitutionNet int64 // 기관 순매수
	IndividualNet int64 // 개인 순매수
}

// FinancialRepository manages financial statement data
type FinancialRepository interface {
	GetLatestByCode(ctx context.Context, code string, date time.Time) (*Financial, error)
	GetByCodeAndQuarter(ctx context.Context, code string, year int, quarter int) (*Financial, error)
	Save(ctx context.Context, financial *Financial) error
	SaveBatch(ctx context.Context, financials []*Financial) error
}

// Financial represents financial metrics
type Financial struct {
	Code      string
	Year      int
	Quarter   int
	Revenue   int64   // 매출액
	OpProfit  int64   // 영업이익
	NetProfit int64   // 순이익
	Assets    int64   // 자산총계
	Equity    int64   // 자본총계
	Debt      int64   // 부채총계
	ROE       float64 // Return on Equity
	DebtRatio float64 // 부채비율
	PER       float64 // Price to Earnings Ratio
	PBR       float64 // Price to Book Ratio
	PSR       float64 // Price to Sales Ratio
}

// DisclosureRepository manages DART disclosure data
type DisclosureRepository interface {
	GetByCodeAndDateRange(ctx context.Context, code string, from, to time.Time) ([]*Disclosure, error)
	GetLatestByCode(ctx context.Context, code string, limit int) ([]*Disclosure, error)
	Save(ctx context.Context, disclosure *Disclosure) error
	SaveBatch(ctx context.Context, disclosures []*Disclosure) error
}

// Disclosure represents a DART disclosure
type Disclosure struct {
	Code    string
	Date    time.Time
	Type    string // 공시 유형
	Title   string
	Content string
}

// SignalRepository manages signal data
type SignalRepository interface {
	GetByDate(ctx context.Context, date time.Time) (*SignalSet, error)
	GetByCodeAndDate(ctx context.Context, code string, date time.Time) (*StockSignals, error)
	Save(ctx context.Context, signalSet *SignalSet) error
}
