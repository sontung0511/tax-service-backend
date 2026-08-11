package validation

import (
	"fmt"
	"strings"
	"time"

	"tax-client/backend/internal/domain"
)

type Issue struct{ TransactionID, Code, Message string }

func Transactions(items []domain.Transaction) []Issue {
	issues := make([]Issue, 0)
	seen := make(map[string]string)
	for _, item := range items {
		if item.ID == "" || item.PeriodID == "" || item.Description == "" || item.RevenueCategory == "" {
			issues = append(issues, Issue{item.ID, "missing", "thiếu mã, kỳ, nội dung hoặc nhóm ngành"})
		}
		if item.Amount <= 0 {
			issues = append(issues, Issue{item.ID, "invalid", "số tiền phải là số nguyên VND dương"})
		}
		if _, err := time.Parse(time.DateOnly, item.Date); err != nil {
			issues = append(issues, Issue{item.ID, "invalid", "ngày phải có dạng YYYY-MM-DD"})
		}
		if item.Type != "revenue" && item.Type != "expense" {
			issues = append(issues, Issue{item.ID, "invalid", "loại giao dịch không hợp lệ"})
		}
		if item.InvoiceNo == "" {
			issues = append(issues, Issue{item.ID, "missing", "thiếu số hóa đơn/chứng từ"})
		}
		key := fmt.Sprintf("%s|%s|%d", item.Date, strings.ToLower(strings.TrimSpace(item.InvoiceNo)), item.Amount)
		if previous, ok := seen[key]; ok {
			issues = append(issues, Issue{item.ID, "duplicate", "trùng giao dịch " + previous})
		} else {
			seen[key] = item.ID
		}
	}
	return issues
}
