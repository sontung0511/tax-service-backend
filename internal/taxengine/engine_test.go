package taxengine

import (
	"testing"
	"time"

	"tax-client/backend/internal/domain"
)

func TestCalculateGroupsRevenueAndUsesIntegerRounding(t *testing.T) {
	period := domain.TaxPeriod{ID: "2026-q3", Year: 2026, PaidAmount: 100}
	items := []domain.Transaction{
		{ID: "1", PeriodID: period.ID, Date: "2026-07-01", Type: "revenue", Amount: 10001, RevenueCategory: "distribution"},
		{ID: "2", PeriodID: period.ID, Date: "2026-07-02", Type: "revenue", Amount: 20000, RevenueCategory: "services"},
		{ID: "3", PeriodID: period.ID, Date: "2026-07-03", Type: "expense", Amount: 5000, RevenueCategory: "distribution"},
	}
	result, err := Calculate(period, items, time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	if result.Revenue != 30001 || result.Expenses != 5000 {
		t.Fatalf("unexpected totals: %+v", result)
	}
	if result.VAT != 1100 || result.PIT != 450 {
		t.Fatalf("unexpected tax: VAT=%d PIT=%d", result.VAT, result.PIT)
	}
	if result.TotalTax != 1550 || result.Remaining != 1450 {
		t.Fatalf("unexpected payable totals: %+v", result)
	}
	if len(result.Lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(result.Lines))
	}
}

func TestCalculateRejectsUnsupportedYear(t *testing.T) {
	_, err := Calculate(domain.TaxPeriod{ID: "p", Year: 2027}, []domain.Transaction{{ID: "1", PeriodID: "p", Date: "2027-01-01", Type: "revenue", Amount: 1, RevenueCategory: "services"}}, time.Now())
	if err == nil {
		t.Fatal("Calculate() expected unsupported rule error")
	}
}

func TestApplyBPSRoundsHalfUp(t *testing.T) {
	got, err := applyBPS(150, 100)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("applyBPS() = %d, want 2", got)
	}
}
