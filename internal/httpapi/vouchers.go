package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tax-client/backend/internal/domain"
	"tax-client/backend/internal/validation"
)

func validVoucherDate(value string) bool {
	_, err := time.Parse(time.DateOnly, value)
	return err == nil
}

func validVoucherStatus(value string) bool { return value == "draft" || value == "saved" }

func validateCashPayment(input domain.CashPayment) error {
	switch {
	case strings.TrimSpace(input.PaymentNo) == "", strings.TrimSpace(input.RecipientName) == "", strings.TrimSpace(input.Description) == "":
		return errors.New("thiếu số phiếu chi, người nhận hoặc diễn giải")
	case !validVoucherDate(input.VoucherDate) || !validVoucherDate(input.PostingDate):
		return errors.New("ngày phiếu và ngày ghi sổ phải có dạng YYYY-MM-DD")
	case !validVoucherStatus(input.Status):
		return errors.New("trạng thái phiếu chi không hợp lệ")
	case input.Amount <= 0:
		return errors.New("số tiền phải là số nguyên VND dương")
	case !validAccount(input.DebitAccount) || !validAccount(input.CreditAccount):
		return errors.New("tài khoản Nợ/Có không hợp lệ")
	}
	return nil
}

func validateCarryForward(input domain.CarryForward, accounts []domain.Account) error {
	if strings.TrimSpace(input.VoucherNo) == "" || strings.TrimSpace(input.Description) == "" {
		return errors.New("thiếu số chứng từ hoặc diễn giải")
	}
	if !validVoucherDate(input.VoucherDate) || !validVoucherDate(input.PostingDate) {
		return errors.New("ngày chứng từ và ngày ghi sổ phải có dạng YYYY-MM-DD")
	}
	if !validVoucherStatus(input.Status) {
		return errors.New("trạng thái chứng từ không hợp lệ")
	}
	if len(input.Entries) == 0 {
		return errors.New("chứng từ kết chuyển cần ít nhất một định khoản")
	}
	for _, entry := range input.Entries {
		if entry.Amount <= 0 {
			return errors.New("số tiền kết chuyển phải là số nguyên VND dương")
		}
		if !validAccount(entry.DebitAccount) || !validAccount(entry.CreditAccount) || entry.DebitAccount == entry.CreditAccount {
			return errors.New("tài khoản Nợ/Có của định khoản không hợp lệ")
		}
		if accounts != nil && (!hasActiveAccount(accounts, entry.DebitAccount) || !hasActiveAccount(accounts, entry.CreditAccount)) {
			return fmt.Errorf("tài khoản %s/%s không tồn tại hoặc đã ngừng sử dụng", entry.DebitAccount, entry.CreditAccount)
		}
	}
	return nil
}

func carryForwardTotal(entries []domain.CarryForwardEntry) domain.Money {
	var total domain.Money
	for _, entry := range entries {
		total += entry.Amount
	}
	return total
}

func cashPaymentTransaction(id string, input domain.CashPayment) (domain.Transaction, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return domain.Transaction{}, err
	}
	invoiceNo := strings.TrimSpace(input.InvoiceNo)
	if invoiceNo == "" {
		invoiceNo = strings.TrimSpace(input.PaymentNo)
	}
	return domain.Transaction{ID: id, PeriodID: input.PeriodID, Date: input.VoucherDate, Type: "expense", Description: strings.TrimSpace(input.Description), InvoiceNo: invoiceNo, DocumentNo: strings.TrimSpace(input.PaymentNo), Amount: input.Amount, RevenueCategory: "other", PaymentStatus: "paid", VoucherType: "cash_payment", CounterpartyCode: strings.TrimSpace(input.RecipientCode), CounterpartyName: strings.TrimSpace(input.RecipientName), CounterpartyTaxCode: strings.TrimSpace(input.RecipientTaxCode), CounterpartyAddress: strings.TrimSpace(input.RecipientAddress), CashPayment: raw}, nil
}

func carryForwardTransaction(id string, input domain.CarryForward) (domain.Transaction, error) {
	for index := range input.Entries {
		if input.Entries[index].ID == "" {
			input.Entries[index].ID = newID("cf")
		}
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return domain.Transaction{}, err
	}
	return domain.Transaction{ID: id, PeriodID: input.PeriodID, Date: input.VoucherDate, Type: "carry_forward", Description: strings.TrimSpace(input.Description), InvoiceNo: strings.TrimSpace(input.VoucherNo), DocumentNo: strings.TrimSpace(input.VoucherNo), Amount: carryForwardTotal(input.Entries), RevenueCategory: "other", VoucherType: "carry_forward", CarryForward: raw}, nil
}

// saveVoucher inserts (existingID == "") or replaces a voucher transaction after the shared period-lock and validation checks.
func (s *Server) saveVoucher(w http.ResponseWriter, r *http.Request, existingID, voucherType, label string, build func(db *domain.Database, id string) (domain.Transaction, func(), error)) {
	id := existingID
	if id == "" {
		id = newID("tx")
	}
	var item domain.Transaction
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		built, afterSave, err := build(db, id)
		if err != nil {
			return validationError{[]validation.Issue{{TransactionID: id, Code: "invalid", Message: err.Error()}}}
		}
		item = built
		target := findPeriod(db.Periods, item.PeriodID)
		if target == nil {
			return errNotFound
		}
		if target.LockedAt != nil {
			return errConflict
		}
		index := -1
		if existingID != "" {
			for i := range db.Transactions {
				if db.Transactions[i].ID == existingID {
					index = i
					break
				}
			}
			if index < 0 || db.Transactions[index].VoucherType != voucherType {
				return errNotFound
			}
			if current := findPeriod(db.Periods, db.Transactions[index].PeriodID); current != nil && current.LockedAt != nil {
				return errConflict
			}
		}
		candidate := append([]domain.Transaction{}, db.Transactions...)
		if index >= 0 {
			candidate[index] = item
		} else {
			candidate = append(candidate, item)
		}
		if issues := validation.Transactions(candidate); len(issues) > 0 {
			return validationError{issues}
		}
		if afterSave != nil {
			afterSave()
		}
		if index >= 0 {
			db.Transactions[index] = item
			prependAudit(db, s.now(), "Cập nhật "+label, item.DocumentNo)
		} else {
			db.Transactions = append([]domain.Transaction{item}, db.Transactions...)
			prependAudit(db, s.now(), "Lập "+label, fmt.Sprintf("%s · %d VND", item.DocumentNo, item.Amount))
		}
		return nil
	})
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "voucher_not_found", "không tìm thấy kỳ kê khai hoặc "+label)
		return
	}
	if errors.Is(err, errConflict) {
		writeError(w, http.StatusConflict, "period_locked", "kỳ kê khai đã khóa")
		return
	}
	var ve validationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "validation_error", "message": label + " không hợp lệ", "issues": ve.issues}})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	status := http.StatusCreated
	if existingID != "" {
		status = http.StatusOK
	}
	writeJSON(w, status, item)
}

func (s *Server) cashPayments(w http.ResponseWriter, r *http.Request) { s.cashPaymentSave(w, r, "") }
func (s *Server) updateCashPayment(w http.ResponseWriter, r *http.Request) {
	s.cashPaymentSave(w, r, r.PathValue("id"))
}

func (s *Server) cashPaymentSave(w http.ResponseWriter, r *http.Request, existingID string) {
	var input domain.CashPayment
	if err := decode(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if existingID != "" && input.ID != "" && input.ID != existingID {
		writeError(w, http.StatusBadRequest, "id_mismatch", "mã phiếu chi trong URL và dữ liệu không khớp")
		return
	}
	s.saveVoucher(w, r, existingID, "cash_payment", "phiếu chi tiền mặt", func(db *domain.Database, id string) (domain.Transaction, func(), error) {
		input.ID = id
		if err := validateCashPayment(input); err != nil {
			return domain.Transaction{}, nil, err
		}
		if !hasActiveAccount(db.Accounts, input.DebitAccount) || !hasActiveAccount(db.Accounts, input.CreditAccount) {
			return domain.Transaction{}, nil, errors.New("tài khoản Nợ/Có không tồn tại hoặc đã ngừng sử dụng")
		}
		item, err := cashPaymentTransaction(id, input)
		save := func() {
			if input.SavePayee && item.CounterpartyCode != "" {
				upsertCounterparty(&db.Counterparties, domain.Counterparty{Code: item.CounterpartyCode, Name: item.CounterpartyName, TaxCode: item.CounterpartyTaxCode, Address: item.CounterpartyAddress})
			}
		}
		return item, save, err
	})
}

func (s *Server) carryForwards(w http.ResponseWriter, r *http.Request) { s.carryForwardSave(w, r, "") }
func (s *Server) updateCarryForward(w http.ResponseWriter, r *http.Request) {
	s.carryForwardSave(w, r, r.PathValue("id"))
}

func (s *Server) carryForwardSave(w http.ResponseWriter, r *http.Request, existingID string) {
	var input domain.CarryForward
	if err := decode(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if existingID != "" && input.ID != "" && input.ID != existingID {
		writeError(w, http.StatusBadRequest, "id_mismatch", "mã chứng từ trong URL và dữ liệu không khớp")
		return
	}
	s.saveVoucher(w, r, existingID, "carry_forward", "chứng từ kết chuyển cuối kỳ", func(db *domain.Database, id string) (domain.Transaction, func(), error) {
		input.ID = id
		if err := validateCarryForward(input, db.Accounts); err != nil {
			return domain.Transaction{}, nil, err
		}
		item, err := carryForwardTransaction(id, input)
		return item, nil, err
	})
}
