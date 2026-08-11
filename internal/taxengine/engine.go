package taxengine

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"tax-client/backend/internal/domain"
)

var ErrUnsupportedRule = errors.New("tax rule is not configured")

type Rule struct {
	ID               string
	Year             int
	Industry         string
	MinAnnualRevenue domain.Money
	MaxAnnualRevenue *domain.Money
	VATRateBPS       int64
	PITRateBPS       int64
}

// Prototype rates. A tax specialist must approve these rules before production use.
var rules = []Rule{
	{ID: "2026-distribution-r1", Year: 2026, Industry: "distribution", VATRateBPS: 100, PITRateBPS: 50},
	{ID: "2026-services-r1", Year: 2026, Industry: "services", VATRateBPS: 500, PITRateBPS: 200},
	{ID: "2026-production-r1", Year: 2026, Industry: "production", VATRateBPS: 300, PITRateBPS: 150},
	{ID: "2026-other-r1", Year: 2026, Industry: "other", VATRateBPS: 200, PITRateBPS: 100},
}

func Calculate(period domain.TaxPeriod, transactions []domain.Transaction, now time.Time) (domain.TaxBreakdown, error) {
	var annualRevenue, revenue, expenses domain.Money
	categoryRevenue := make(map[string]domain.Money)
	for _, item := range transactions {
		date, err := time.Parse(time.DateOnly, item.Date)
		if err != nil {
			return domain.TaxBreakdown{}, fmt.Errorf("transaction %s date: %w", item.ID, err)
		}
		if item.Type == "revenue" && date.Year() == period.Year {
			annualRevenue += item.Amount
		}
		if item.PeriodID != period.ID {
			continue
		}
		switch item.Type {
		case "revenue":
			revenue += item.Amount
			categoryRevenue[item.RevenueCategory] += item.Amount
		case "expense":
			expenses += item.Amount
		default:
			return domain.TaxBreakdown{}, fmt.Errorf("transaction %s has invalid type", item.ID)
		}
	}
	result := domain.TaxBreakdown{PeriodID: period.ID, Revenue: revenue, Expenses: expenses, Paid: period.PaidAmount, Lines: []domain.TaxLine{}, CalculatedAt: now.UTC()}
	versions := make([]string, 0, len(categoryRevenue))
	industries := make([]string, 0, len(categoryRevenue))
	for industry := range categoryRevenue {
		industries = append(industries, industry)
	}
	sort.Strings(industries)
	for _, industry := range industries {
		amount := categoryRevenue[industry]
		rule, err := findRule(period.Year, industry, annualRevenue)
		if err != nil {
			return domain.TaxBreakdown{}, err
		}
		vat, err := applyBPS(amount, rule.VATRateBPS)
		if err != nil {
			return domain.TaxBreakdown{}, err
		}
		pit, err := applyBPS(amount, rule.PITRateBPS)
		if err != nil {
			return domain.TaxBreakdown{}, err
		}
		result.Lines = append(result.Lines, domain.TaxLine{Industry: industry, Revenue: amount, VATRateBPS: rule.VATRateBPS, PITRateBPS: rule.PITRateBPS, VAT: vat, PIT: pit, RuleID: rule.ID})
		result.VAT += vat
		result.PIT += pit
		versions = append(versions, rule.ID)
	}
	if len(result.Lines) > 0 {
		result.VATRateBPS = result.Lines[0].VATRateBPS
		result.PITRateBPS = result.Lines[0].PITRateBPS
	}
	result.TotalTax = result.VAT + result.PIT
	if result.TotalTax > result.Paid {
		result.Remaining = result.TotalTax - result.Paid
	}
	if len(versions) == 0 {
		result.FormulaVersion = fmt.Sprintf("%d-no-revenue", period.Year)
	} else {
		result.FormulaVersion = strings.Join(versions, "+")
	}
	return result, nil
}

func findRule(year int, industry string, annualRevenue domain.Money) (Rule, error) {
	for _, rule := range rules {
		if rule.Year == year && rule.Industry == industry && annualRevenue >= rule.MinAnnualRevenue && (rule.MaxAnnualRevenue == nil || annualRevenue <= *rule.MaxAnnualRevenue) {
			return rule, nil
		}
	}
	return Rule{}, fmt.Errorf("%w: year=%d industry=%s revenue=%d", ErrUnsupportedRule, year, industry, annualRevenue)
}

func applyBPS(amount domain.Money, bps int64) (domain.Money, error) {
	if amount < 0 || bps < 0 || bps > 10000 {
		return 0, errors.New("invalid monetary amount or basis points")
	}
	quotient, remainder := int64(amount)/10000, int64(amount)%10000
	return domain.Money(quotient*bps + (remainder*bps+5000)/10000), nil
}
