package validation

import (
	"testing"

	"tax-client/backend/internal/domain"
)

func TestTransactionsDetectsDuplicate(t *testing.T) {
	items := []domain.Transaction{
		{ID: "a", PeriodID: "p", Date: "2026-01-01", Type: "revenue", Description: "A", InvoiceNo: "HD-1", Amount: 100, RevenueCategory: "distribution"},
		{ID: "b", PeriodID: "p", Date: "2026-01-01", Type: "revenue", Description: "B", InvoiceNo: " hd-1 ", Amount: 100, RevenueCategory: "distribution"},
	}
	issues := Transactions(items)
	if len(issues) != 1 || issues[0].Code != "duplicate" || issues[0].TransactionID != "b" {
		t.Fatalf("unexpected issues: %+v", issues)
	}
}

func TestTransactionsReportsInvalidFields(t *testing.T) {
	issues := Transactions([]domain.Transaction{{ID: "bad", Date: "not-a-date", Type: "wrong", Amount: -1}})
	if len(issues) < 4 {
		t.Fatalf("issues = %d, want at least 4: %+v", len(issues), issues)
	}
}
