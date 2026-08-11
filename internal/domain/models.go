package domain

import "time"

type Money int64

type BusinessProfile struct {
	TaxpayerType        string `json:"taxpayerType"`
	TaxCode             string `json:"taxCode"`
	BusinessName        string `json:"businessName"`
	OwnerName           string `json:"ownerName"`
	Address             string `json:"address"`
	Industry            string `json:"industry"`
	DeclarationKind     string `json:"declarationKind"`
	EnterpriseType      string `json:"enterpriseType,omitempty"`
	LegalRepresentative string `json:"legalRepresentative,omitempty"`
	VATMethod           string `json:"vatMethod,omitempty"`
	AccountingRegime    string `json:"accountingRegime,omitempty"`
	HouseholdTaxMethod  string `json:"householdTaxMethod,omitempty"`
}

type TaxPeriod struct {
	ID          string        `json:"id"`
	Label       string        `json:"label"`
	Year        int           `json:"year"`
	Kind        string        `json:"kind"`
	Status      string        `json:"status"`
	DueDate     string        `json:"dueDate"`
	PaidAmount  Money         `json:"paidAmount"`
	LockedAt    *time.Time    `json:"lockedAt,omitempty"`
	TaxSnapshot *TaxBreakdown `json:"taxSnapshot,omitempty"`
}

type Transaction struct {
	ID                string `json:"id"`
	PeriodID          string `json:"periodId"`
	Date              string `json:"date"`
	Type              string `json:"type"`
	Description       string `json:"description"`
	InvoiceNo         string `json:"invoiceNo"`
	Amount            Money  `json:"amount"`
	RevenueCategory   string `json:"revenueCategory"`
	DocumentNo        string `json:"documentNo,omitempty"`
	PaymentStatus     string `json:"paymentStatus,omitempty"`
	OutstandingAmount Money  `json:"outstandingAmount,omitempty"`
}

type TaxLine struct {
	Industry   string `json:"industry"`
	Revenue    Money  `json:"revenue"`
	VATRateBPS int64  `json:"vatRateBps"`
	PITRateBPS int64  `json:"pitRateBps"`
	VAT        Money  `json:"vat"`
	PIT        Money  `json:"pit"`
	RuleID     string `json:"ruleId"`
}

type TaxBreakdown struct {
	PeriodID       string    `json:"periodId"`
	Revenue        Money     `json:"revenue"`
	Expenses       Money     `json:"expenses"`
	VATRateBPS     int64     `json:"vatRateBps"`
	PITRateBPS     int64     `json:"pitRateBps"`
	VAT            Money     `json:"vat"`
	PIT            Money     `json:"pit"`
	TotalTax       Money     `json:"totalTax"`
	Paid           Money     `json:"paid"`
	Remaining      Money     `json:"remaining"`
	FormulaVersion string    `json:"formulaVersion"`
	Lines          []TaxLine `json:"lines"`
	CalculatedAt   time.Time `json:"calculatedAt"`
}

type TaxDeclaration struct {
	ID            string           `json:"id"`
	TaxType       string           `json:"taxType"`
	FormCode      string           `json:"formCode"`
	FormName      string           `json:"formName"`
	PeriodLabel   string           `json:"periodLabel"`
	PeriodType    string           `json:"periodType"`
	Status        string           `json:"status"`
	SchemaVersion string           `json:"schemaVersion"`
	Values        map[string]Money `json:"values"`
	UpdatedAt     time.Time        `json:"updatedAt"`
	LockedAt      *time.Time       `json:"lockedAt,omitempty"`
}

type AuditEntry struct {
	ID     string    `json:"id"`
	At     time.Time `json:"at"`
	Action string    `json:"action"`
	Detail string    `json:"detail"`
}

type Database struct {
	Profile      BusinessProfile  `json:"profile"`
	Periods      []TaxPeriod      `json:"periods"`
	Transactions []Transaction    `json:"transactions"`
	Declarations []TaxDeclaration `json:"declarations"`
	Audit        []AuditEntry     `json:"audit"`
}
