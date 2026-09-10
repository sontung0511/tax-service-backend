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
	ID                  string           `json:"id"`
	PeriodID            string           `json:"periodId"`
	Date                string           `json:"date"`
	Type                string           `json:"type"`
	Description         string           `json:"description"`
	InvoiceNo           string           `json:"invoiceNo"`
	Amount              Money            `json:"amount"`
	VATAmount           Money            `json:"vatAmount"`
	RevenueCategory     string           `json:"revenueCategory"`
	DocumentNo          string           `json:"documentNo,omitempty"`
	PaymentStatus       string           `json:"paymentStatus,omitempty"`
	OutstandingAmount   Money            `json:"outstandingAmount,omitempty"`
	VoucherType         string           `json:"voucherType,omitempty"`
	CounterpartyCode    string           `json:"counterpartyCode,omitempty"`
	CounterpartyName    string           `json:"counterpartyName,omitempty"`
	CounterpartyTaxCode string           `json:"counterpartyTaxCode,omitempty"`
	CounterpartyAddress string           `json:"counterpartyAddress,omitempty"`
	CashReceipt         *CashReceiptData `json:"cashReceipt,omitempty"`
}

// Counterparty is a reusable customer or supplier record used on vouchers.
type Counterparty struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	TaxCode string `json:"taxCode"`
	Address string `json:"address"`
}

type Account struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	ParentCode string `json:"parentCode,omitempty"`
	IsActive   bool   `json:"isActive"`
}

// CashReceipt contains the data required to create a cash receipt voucher.
type CashReceipt struct {
	ID                  string                   `json:"id,omitempty"`
	PeriodID            string                   `json:"periodId"`
	VoucherDate         string                   `json:"voucherDate"`
	AccountingDate      string                   `json:"accountingDate"`
	Status              string                   `json:"status"`
	ReceiptNo           string                   `json:"receiptNo"`
	CounterpartyCode    string                   `json:"counterpartyCode"`
	CounterpartyName    string                   `json:"counterpartyName"`
	CounterpartyTaxCode string                   `json:"counterpartyTaxCode"`
	CounterpartyAddress string                   `json:"counterpartyAddress"`
	Description         string                   `json:"description"`
	ContactName         string                   `json:"contactName"`
	DebitAccount        string                   `json:"debitAccount"`
	CreditAccount       string                   `json:"creditAccount"`
	Amount              Money                    `json:"amount"`
	AmountIncludesVAT   bool                     `json:"amountIncludesVAT"`
	Currency            string                   `json:"currency"`
	ExchangeRate        float64                  `json:"exchangeRate"`
	ConvertedAmount     Money                    `json:"convertedAmount"`
	RevenueCategory     string                   `json:"revenueCategory"`
	InvoiceNo           string                   `json:"invoiceNo"`
	InvoiceSymbol       string                   `json:"invoiceSymbol"`
	InvoiceDate         string                   `json:"invoiceDate"`
	DetailCode          string                   `json:"detailCode"`
	Quantity            string                   `json:"quantity"`
	UnitPrice           Money                    `json:"unitPrice"`
	CaseCode            string                   `json:"caseCode"`
	Collector           string                   `json:"collector"`
	Note                string                   `json:"note"`
	Attachments         []ReceiptAttachment      `json:"attachments"`
	Entries             []ReceiptAccountingEntry `json:"entries"`
	Invoices            []ReceiptInvoice         `json:"invoices"`
	TaxLines            []ReceiptTaxLine         `json:"taxLines"`
	SaveCounterparty    bool                     `json:"saveCounterparty"`
}

type ReceiptAttachment struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
	Data string `json:"data"`
}

type ReceiptAccountingEntry struct {
	ID            string  `json:"id"`
	VoucherID     string  `json:"voucherId"`
	InvoiceID     string  `json:"invoiceId"`
	DebitAccount  string  `json:"debitAccount"`
	CreditAccount string  `json:"creditAccount"`
	Amount        Money   `json:"amount"`
	Description   string  `json:"description"`
	Kind          string  `json:"kind"`
	Rate          float64 `json:"rate,omitempty"`
	DetailCode    string  `json:"detailCode,omitempty"`
	Quantity      string  `json:"quantity,omitempty"`
	UnitPrice     Money   `json:"unitPrice,omitempty"`
	RevenueDetailID string `json:"revenueDetailId,omitempty"`
}

type ReceiptInvoice struct {
	ID          string `json:"id"`
	VoucherID   string `json:"voucherId"`
	InvoiceNo   string `json:"invoiceNo"`
	Symbol      string `json:"symbol"`
	InvoiceDate string `json:"invoiceDate"`
	TaxCode     string `json:"taxCode"`
}

type ReceiptTaxLine struct {
	ID               string  `json:"id"`
	VoucherID        string  `json:"voucherId"`
	InvoiceID        string  `json:"invoiceId"`
	RevenueDetailID  string  `json:"revenueDetailId"`
	TaxRate          float64 `json:"taxRate"`
	TaxableAmount    Money   `json:"taxableAmount"`
	TaxAmount        Money   `json:"taxAmount"`
	PriceIncludesTax bool    `json:"priceIncludesTax"`
}

type CashReceiptData struct {
	VoucherDate       string                   `json:"voucherDate"`
	AccountingDate    string                   `json:"accountingDate"`
	Status            string                   `json:"status"`
	ContactName       string                   `json:"contactName"`
	DebitAccount      string                   `json:"debitAccount"`
	CreditAccount     string                   `json:"creditAccount"`
	Currency          string                   `json:"currency"`
	ExchangeRate      float64                  `json:"exchangeRate"`
	ConvertedAmount   Money                    `json:"convertedAmount"`
	AmountIncludesVAT bool                     `json:"amountIncludesVAT"`
	InvoiceNo         string                   `json:"invoiceNo"`
	InvoiceSymbol     string                   `json:"invoiceSymbol"`
	InvoiceDate       string                   `json:"invoiceDate"`
	DetailCode        string                   `json:"detailCode"`
	Quantity          string                   `json:"quantity"`
	UnitPrice         Money                    `json:"unitPrice"`
	CaseCode          string                   `json:"caseCode"`
	Collector         string                   `json:"collector"`
	Note              string                   `json:"note"`
	Attachments       []ReceiptAttachment      `json:"attachments"`
	Entries           []ReceiptAccountingEntry `json:"entries"`
	Invoices          []ReceiptInvoice         `json:"invoices"`
	TaxLines          []ReceiptTaxLine         `json:"taxLines"`
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
	Profile        BusinessProfile  `json:"profile"`
	Periods        []TaxPeriod      `json:"periods"`
	Transactions   []Transaction    `json:"transactions"`
	Counterparties []Counterparty   `json:"counterparties"`
	Accounts       []Account        `json:"accounts"`
	Declarations   []TaxDeclaration `json:"declarations"`
	Audit          []AuditEntry     `json:"audit"`
}
