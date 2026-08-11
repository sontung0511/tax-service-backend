package seed

import (
	"time"

	"tax-client/backend/internal/domain"
)

func Database() domain.Database {
	return domain.Database{
		Profile: domain.BusinessProfile{TaxpayerType: "household", TaxCode: "0312345678", BusinessName: "Hộ kinh doanh An Nhiên", OwnerName: "Nguyễn Minh An", Address: "24 Nguyễn Văn Trỗi, Phú Nhuận, TP.HCM", Industry: "distribution", DeclarationKind: "quarter", HouseholdTaxMethod: "revenue_percentage"},
		Periods: []domain.TaxPeriod{
			{ID: "2026-q3", Label: "Quý 3/2026", Year: 2026, Kind: "quarter", Status: "draft", DueDate: "2026-10-31"},
			{ID: "2026-q2", Label: "Quý 2/2026", Year: 2026, Kind: "quarter", Status: "draft", DueDate: "2026-07-31"},
			{ID: "2026-q1", Label: "Quý 1/2026", Year: 2026, Kind: "quarter", Status: "draft", DueDate: "2026-04-30"},
		},
		Transactions: []domain.Transaction{},
		Declarations: []domain.TaxDeclaration{
			{ID: "vat-2026-q3", TaxType: "vat", FormCode: "01/GTGT", FormName: "Tờ khai thuế giá trị gia tăng", PeriodLabel: "Quý 3/2026", PeriodType: "quarter", Status: "draft", SchemaVersion: "TT80-01GTGT-draft", Values: map[string]domain.Money{"outputVat": 0, "deductibleInputVat": 0, "payableVat": 0}, UpdatedAt: time.Now().UTC()},
			{ID: "pit-2026-q3", TaxType: "pit", FormCode: "05/KK-TNCN", FormName: "Tờ khai khấu trừ thuế thu nhập cá nhân", PeriodLabel: "Quý 3/2026", PeriodType: "quarter", Status: "draft", SchemaVersion: "TT80-05KKTNCN-draft", Values: map[string]domain.Money{"employees": 0, "taxableIncome": 0, "withheldTax": 0}, UpdatedAt: time.Now().UTC()},
			{ID: "cit-2026", TaxType: "cit", FormCode: "03/TNDN", FormName: "Tờ khai quyết toán thuế thu nhập doanh nghiệp", PeriodLabel: "Năm 2026", PeriodType: "year", Status: "draft", SchemaVersion: "TT80-03TNDN-draft", Values: map[string]domain.Money{"revenue": 0, "deductibleExpenses": 0, "taxableIncome": 0, "payableTax": 0}, UpdatedAt: time.Now().UTC()},
		},
		Audit: []domain.AuditEntry{},
	}
}
