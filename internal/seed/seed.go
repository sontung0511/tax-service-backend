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
		Counterparties: []domain.Counterparty{
			{Code: "KH-001", Name: "Công ty TNHH Thương mại Minh Long", TaxCode: "0312345678", Address: "12 Nguyễn Huệ, Phường Sài Gòn, TP.HCM"},
			{Code: "KH-002", Name: "Nguyễn Thị Lan", TaxCode: "", Address: "45 Lê Văn Sỹ, Phường Nhiêu Lộc, TP.HCM"},
		},
		Accounts: []domain.Account{
			{Code: "111", Name: "Tiền mặt", IsActive: true}, {Code: "1111", Name: "- Tiền Việt Nam", ParentCode: "111", IsActive: true}, {Code: "112", Name: "Tiền gửi ngân hàng", IsActive: true}, {Code: "1121", Name: "- Tiền Việt Nam", ParentCode: "112", IsActive: true}, {Code: "131", Name: "Phải thu của khách hàng", IsActive: true}, {Code: "133", Name: "Thuế GTGT được khấu trừ", IsActive: true}, {Code: "1331", Name: "- Thuế GTGT được khấu trừ của HH-DV", ParentCode: "133", IsActive: true}, {Code: "152", Name: "Nguyên liệu, vật liệu", IsActive: true}, {Code: "154", Name: "Chi phí SXKD dở dang", IsActive: true}, {Code: "156", Name: "Hàng hóa", IsActive: true}, {Code: "1561", Name: "- Giá mua hàng hóa", ParentCode: "156", IsActive: true}, {Code: "211", Name: "Tài sản cố định", IsActive: true}, {Code: "2111", Name: "- TSCĐ hữu hình", ParentCode: "211", IsActive: true}, {Code: "214", Name: "Hao mòn TSCĐ", IsActive: true}, {Code: "2141", Name: "- Hao mòn TSCĐ hữu hình", ParentCode: "214", IsActive: true}, {Code: "242", Name: "Chi phí trả trước dài hạn", IsActive: true}, {Code: "331", Name: "Phải trả cho người bán", IsActive: true}, {Code: "333", Name: "Thuế và các khoản phải nộp Nhà nước", IsActive: true}, {Code: "3331", Name: "- Thuế GTGT phải nộp", ParentCode: "333", IsActive: true}, {Code: "3334", Name: "- Thuế thu nhập doanh nghiệp", ParentCode: "333", IsActive: true}, {Code: "334", Name: "Phải trả người lao động", IsActive: true}, {Code: "338", Name: "Phải trả, phải nộp khác", IsActive: true}, {Code: "3382", Name: "- Kinh phí công đoàn", ParentCode: "338", IsActive: true}, {Code: "3383", Name: "- Bảo hiểm xã hội", ParentCode: "338", IsActive: true}, {Code: "411", Name: "Nguồn vốn kinh doanh", IsActive: true}, {Code: "4111", Name: "- Vốn đầu tư của chủ sở hữu", ParentCode: "411", IsActive: true}, {Code: "418", Name: "Các quỹ thuộc vốn chủ sở hữu", IsActive: true}, {Code: "421", Name: "Lợi nhuận chưa phân phối", IsActive: true}, {Code: "4211", Name: "- Lợi nhuận sau thuế chưa phân phối năm trước", ParentCode: "421", IsActive: true}, {Code: "511", Name: "Doanh thu bán hàng và cung cấp dịch vụ", IsActive: true}, {Code: "5111", Name: "- Doanh thu bán Hàng hóa", ParentCode: "511", IsActive: true}, {Code: "5113", Name: "- Doanh thu cung cấp Dịch vụ", ParentCode: "511", IsActive: true}, {Code: "515", Name: "Doanh thu hoạt động tài chính", IsActive: true}, {Code: "632", Name: "Giá vốn bán hàng", IsActive: true}, {Code: "6321", Name: "- Giá vốn bán hàng Hàng Hóa", ParentCode: "632", IsActive: true}, {Code: "6323", Name: "- Giá vốn bán hàng Dịch vụ", ParentCode: "632", IsActive: true}, {Code: "642", Name: "Chi phí quản lý kinh doanh", IsActive: true}, {Code: "6422", Name: "- Chi phí quản lý doanh nghiệp", ParentCode: "642", IsActive: true}, {Code: "911", Name: "Xác định kết quả kinh doanh", IsActive: true},
		},
		Declarations: []domain.TaxDeclaration{
			{ID: "vat-2026-q3", TaxType: "vat", FormCode: "01/GTGT", FormName: "Tờ khai thuế giá trị gia tăng", PeriodLabel: "Quý 3/2026", PeriodType: "quarter", Status: "draft", SchemaVersion: "TT80-01GTGT-draft", Values: map[string]domain.Money{"outputVat": 0, "deductibleInputVat": 0, "payableVat": 0}, UpdatedAt: time.Now().UTC()},
			{ID: "pit-2026-q3", TaxType: "pit", FormCode: "05/KK-TNCN", FormName: "Tờ khai khấu trừ thuế thu nhập cá nhân", PeriodLabel: "Quý 3/2026", PeriodType: "quarter", Status: "draft", SchemaVersion: "TT80-05KKTNCN-draft", Values: map[string]domain.Money{"employees": 0, "taxableIncome": 0, "withheldTax": 0}, UpdatedAt: time.Now().UTC()},
			{ID: "cit-2026", TaxType: "cit", FormCode: "03/TNDN", FormName: "Tờ khai quyết toán thuế thu nhập doanh nghiệp", PeriodLabel: "Năm 2026", PeriodType: "year", Status: "draft", SchemaVersion: "TT80-03TNDN-draft", Values: map[string]domain.Money{"revenue": 0, "deductibleExpenses": 0, "taxableIncome": 0, "payableTax": 0}, UpdatedAt: time.Now().UTC()},
		},
		Audit: []domain.AuditEntry{},
	}
}
