package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tax-client/backend/internal/domain"
	"tax-client/backend/internal/repository"
	"tax-client/backend/internal/taxengine"
	"tax-client/backend/internal/validation"
)

const maxBodyBytes = 10 << 20

var (
	errNotFound = errors.New("resource not found")
	errConflict = errors.New("resource conflict")
)

type Config struct {
	AllowedOrigin string
	Token         string
	Logger        *slog.Logger
	Now           func() time.Time
}

type Server struct {
	repo          repository.Repository
	allowedOrigin string
	token         string
	logger        *slog.Logger
	now           func() time.Time
}

func New(repo repository.Repository, cfg Config) http.Handler {
	if cfg.AllowedOrigin == "" {
		cfg.AllowedOrigin = "http://localhost:3000"
	}
	if cfg.Token == "" {
		cfg.Token = "mock-session-token"
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	s := &Server{repo: repo, allowedOrigin: cfg.AllowedOrigin, token: cfg.Token, logger: cfg.Logger, now: cfg.Now}

	public := http.NewServeMux()
	public.HandleFunc("GET /healthz", s.health)
	public.HandleFunc("POST /api/login", s.login)

	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/database", s.getDatabase)
	protected.HandleFunc("GET /api/profile", s.profile)
	protected.HandleFunc("PUT /api/profile", s.profile)
	protected.HandleFunc("GET /api/tax-periods", s.taxPeriods)
	protected.HandleFunc("POST /api/tax-periods", s.taxPeriods)
	protected.HandleFunc("POST /api/tax-periods/{id}/lock", s.lockPeriod)
	protected.HandleFunc("GET /api/transactions", s.transactions)
	protected.HandleFunc("POST /api/transactions", s.transactions)
	protected.HandleFunc("PUT /api/transactions/{id}", s.updateTransaction)
	protected.HandleFunc("DELETE /api/transactions/{id}", s.deleteTransaction)
	protected.HandleFunc("GET /api/counterparties", s.counterparties)
	protected.HandleFunc("GET /api/accounts", s.accounts)
	protected.HandleFunc("GET /api/counterparties/{code}", s.counterparty)
	protected.HandleFunc("PUT /api/counterparties/{code}", s.counterparty)
	protected.HandleFunc("POST /api/cash-receipts", s.cashReceipts)
	protected.HandleFunc("PUT /api/cash-receipts/{id}", s.updateCashReceipt)
	protected.HandleFunc("POST /api/calculate", s.calculate)
	protected.HandleFunc("POST /api/imports", s.importTransactions)
	protected.HandleFunc("GET /api/exports", s.exportData)
	protected.HandleFunc("GET /api/declarations", s.declarations)
	protected.HandleFunc("PUT /api/declarations/{id}", s.declarations)
	protected.HandleFunc("GET /api/audit", s.audit)

	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/healthz" || r.URL.Path == "/api/login" {
			public.ServeHTTP(w, r)
			return
		}
		s.auth(protected).ServeHTTP(w, r)
	})
	return s.recover(s.cors(s.log(root)))
}

func (s *Server) accounts(w http.ResponseWriter, r *http.Request) {
	db, err := s.repo.Snapshot(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
	items := make([]domain.Account, 0, len(db.Accounts))
	for _, item := range db.Accounts {
		if item.IsActive && (query == "" || strings.Contains(strings.ToLower(item.Code), query) || strings.Contains(strings.ToLower(item.Name), query)) {
			items = append(items, item)
		}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decode(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if subtle.ConstantTimeCompare([]byte(input.Username), []byte("demo")) != 1 || subtle.ConstantTimeCompare([]byte(input.Password), []byte("demo123")) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "tài khoản hoặc mật khẩu không đúng")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": s.token, "displayName": "Nguyễn Minh An"})
}

func (s *Server) getDatabase(w http.ResponseWriter, r *http.Request) {
	db, err := s.repo.Snapshot(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, db)
}

func (s *Server) profile(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		db, err := s.repo.Snapshot(r.Context())
		if err != nil {
			s.internal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, db.Profile)
		return
	}
	var profile domain.BusinessProfile
	if err := decode(w, r, &profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := validateProfile(profile); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		db.Profile = profile
		prependAudit(db, s.now(), "Cập nhật hồ sơ", profile.BusinessName)
		return nil
	})
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) taxPeriods(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		db, err := s.repo.Snapshot(r.Context())
		if err != nil {
			s.internal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, db.Periods)
		return
	}
	var period domain.TaxPeriod
	if err := decode(w, r, &period); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if period.ID == "" {
		period.ID = newID("period")
	}
	if err := validatePeriod(period); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		for _, item := range db.Periods {
			if item.ID == period.ID {
				return errConflict
			}
		}
		db.Periods = append([]domain.TaxPeriod{period}, db.Periods...)
		prependAudit(db, s.now(), "Tạo kỳ kê khai", period.Label)
		return nil
	})
	if errors.Is(err, errConflict) {
		writeError(w, http.StatusConflict, "period_exists", "kỳ kê khai đã tồn tại")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, period)
}

func (s *Server) lockPeriod(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var locked domain.TaxPeriod
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		period := findPeriod(db.Periods, id)
		if period == nil {
			return errNotFound
		}
		if period.LockedAt != nil {
			locked = *period
			return nil
		}
		periodItems := filterTransactions(db.Transactions, id)
		if issues := validation.Transactions(periodItems); len(issues) > 0 {
			return validationError{issues}
		}
		result, err := taxengine.Calculate(*period, db.Transactions, s.now())
		if err != nil {
			return err
		}
		now := s.now().UTC()
		period.Status = "completed"
		period.LockedAt = &now
		period.TaxSnapshot = &result
		locked = *period
		prependAudit(db, now, "Khóa kỳ kê khai", fmt.Sprintf("%s · %s", period.Label, result.FormulaVersion))
		return nil
	})
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "period_not_found", "không tìm thấy kỳ kê khai")
		return
	}
	var ve validationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "invalid_transactions", "message": "dữ liệu giao dịch chưa hợp lệ", "issues": ve.issues}})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, locked)
}

func (s *Server) transactions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		db, err := s.repo.Snapshot(r.Context())
		if err != nil {
			s.internal(w, err)
			return
		}
		items := db.Transactions
		if periodID := r.URL.Query().Get("periodId"); periodID != "" {
			items = filterTransactions(items, periodID)
		}
		writeJSON(w, http.StatusOK, items)
		return
	}
	var item domain.Transaction
	if err := decode(w, r, &item); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if item.ID == "" {
		item.ID = newID("tx")
	}
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		period := findPeriod(db.Periods, item.PeriodID)
		if period == nil {
			return errNotFound
		}
		if period.LockedAt != nil {
			return errConflict
		}
		candidate := append(append([]domain.Transaction{}, db.Transactions...), item)
		if issues := validation.Transactions(candidate); len(issues) > 0 {
			return validationError{issues}
		}
		db.Transactions = append([]domain.Transaction{item}, db.Transactions...)
		prependAudit(db, s.now(), "Cập nhật giao dịch", fmt.Sprintf("%s · %d VND", item.Description, item.Amount))
		return nil
	})
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "period_not_found", "không tìm thấy kỳ kê khai")
		return
	}
	if errors.Is(err, errConflict) {
		writeError(w, http.StatusConflict, "period_locked", "kỳ kê khai đã khóa")
		return
	}
	var ve validationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "validation_error", "message": "giao dịch không hợp lệ", "issues": ve.issues}})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) counterparties(w http.ResponseWriter, r *http.Request) {
	db, err := s.repo.Snapshot(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
	items := make([]domain.Counterparty, 0, len(db.Counterparties))
	for _, item := range db.Counterparties {
		if query == "" || strings.Contains(strings.ToLower(item.Code), query) || strings.Contains(strings.ToLower(item.Name), query) {
			items = append(items, item)
		}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) counterparty(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.PathValue("code"))
	if r.Method == http.MethodGet {
		db, err := s.repo.Snapshot(r.Context())
		if err != nil {
			s.internal(w, err)
			return
		}
		for _, item := range db.Counterparties {
			if item.Code == code {
				writeJSON(w, http.StatusOK, item)
				return
			}
		}
		writeError(w, http.StatusNotFound, "counterparty_not_found", "không tìm thấy đối tượng")
		return
	}

	var input domain.Counterparty
	if err := decode(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if input.Code != "" && input.Code != code {
		writeError(w, http.StatusBadRequest, "id_mismatch", "mã đối tượng trong URL và dữ liệu không khớp")
		return
	}
	input.Code = code
	if err := validateCounterparty(input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	if err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		upsertCounterparty(&db.Counterparties, input)
		prependAudit(db, s.now(), "Cập nhật danh mục đối tượng", input.Code+" · "+input.Name)
		return nil
	}); err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, input)
}

func (s *Server) cashReceipts(w http.ResponseWriter, r *http.Request) {
	var input domain.CashReceipt
	if err := decode(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	voucherID := newID("tx")
	normalizeCashReceipt(voucherID, &input)
	if err := validateCashReceipt(input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	item := receiptTransaction(voucherID, input)
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		period := findPeriod(db.Periods, item.PeriodID)
		if period == nil {
			return errNotFound
		}
		if period.LockedAt != nil {
			return errConflict
		}
		if err := validateReceiptAccountCatalog(db.Accounts, item.CashReceipt); err != nil {
			return validationError{[]validation.Issue{{TransactionID: item.ID, Code: "invalid_account", Message: err.Error()}}}
		}
		candidate := append(append([]domain.Transaction{}, db.Transactions...), item)
		if issues := validation.Transactions(candidate); len(issues) > 0 {
			return validationError{issues}
		}
		if input.SaveCounterparty {
			upsertCounterparty(&db.Counterparties, domain.Counterparty{Code: item.CounterpartyCode, Name: item.CounterpartyName, TaxCode: item.CounterpartyTaxCode, Address: item.CounterpartyAddress})
		}
		db.Transactions = append([]domain.Transaction{item}, db.Transactions...)
		prependAudit(db, s.now(), "Lập phiếu thu tiền mặt", fmt.Sprintf("%s · %d VND", item.DocumentNo, item.Amount))
		return nil
	})
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "period_not_found", "không tìm thấy kỳ kê khai")
		return
	}
	if errors.Is(err, errConflict) {
		writeError(w, http.StatusConflict, "period_locked", "kỳ kê khai đã khóa")
		return
	}
	var ve validationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "validation_error", "message": "phiếu thu không hợp lệ", "issues": ve.issues}})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) updateCashReceipt(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var input domain.CashReceipt
	if err := decode(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if input.ID != "" && input.ID != id {
		writeError(w, http.StatusBadRequest, "id_mismatch", "mã phiếu thu trong URL và dữ liệu không khớp")
		return
	}
	normalizeCashReceipt(id, &input)
	if err := validateCashReceipt(input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	item := receiptTransaction(id, input)
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		for index, current := range db.Transactions {
			if current.ID != id {
				continue
			}
			if current.VoucherType != "cash_receipt" {
				return errNotFound
			}
			period := findPeriod(db.Periods, current.PeriodID)
			target := findPeriod(db.Periods, item.PeriodID)
			if target == nil {
				return errNotFound
			}
			if (period != nil && period.LockedAt != nil) || target.LockedAt != nil {
				return errConflict
			}
			if err := validateReceiptAccountCatalog(db.Accounts, item.CashReceipt); err != nil {
				return validationError{[]validation.Issue{{TransactionID: item.ID, Code: "invalid_account", Message: err.Error()}}}
			}
			candidate := append([]domain.Transaction{}, db.Transactions...)
			candidate[index] = item
			if issues := validation.Transactions(candidate); len(issues) > 0 {
				return validationError{issues}
			}
			if input.SaveCounterparty {
				upsertCounterparty(&db.Counterparties, domain.Counterparty{Code: item.CounterpartyCode, Name: item.CounterpartyName, TaxCode: item.CounterpartyTaxCode, Address: item.CounterpartyAddress})
			}
			db.Transactions[index] = item
			prependAudit(db, s.now(), "Cập nhật phiếu thu tiền mặt", item.DocumentNo)
			return nil
		}
		return errNotFound
	})
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "cash_receipt_not_found", "không tìm thấy phiếu thu")
		return
	}
	if errors.Is(err, errConflict) {
		writeError(w, http.StatusConflict, "period_locked", "kỳ kê khai đã khóa")
		return
	}
	var ve validationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "validation_error", "message": "phiếu thu không hợp lệ", "issues": ve.issues}})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		for index, item := range db.Transactions {
			if item.ID != id {
				continue
			}
			period := findPeriod(db.Periods, item.PeriodID)
			if period != nil && period.LockedAt != nil {
				return errConflict
			}
			db.Transactions = append(db.Transactions[:index], db.Transactions[index+1:]...)
			prependAudit(db, s.now(), "Xóa giao dịch", fmt.Sprintf("%s · %d VND", item.Description, item.Amount))
			return nil
		}
		return errNotFound
	})
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "transaction_not_found", "không tìm thấy giao dịch")
		return
	}
	if errors.Is(err, errConflict) {
		writeError(w, http.StatusConflict, "period_locked", "không thể xóa giao dịch của kỳ đã khóa")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateTransaction(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var item domain.Transaction
	if err := decode(w, r, &item); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if item.ID != "" && item.ID != id {
		writeError(w, http.StatusBadRequest, "id_mismatch", "mã giao dịch trong đường dẫn và dữ liệu không khớp")
		return
	}
	item.ID = id
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		index := -1
		for i := range db.Transactions {
			if db.Transactions[i].ID == id {
				index = i
				break
			}
		}
		if index < 0 {
			return errNotFound
		}
		currentPeriod := findPeriod(db.Periods, db.Transactions[index].PeriodID)
		targetPeriod := findPeriod(db.Periods, item.PeriodID)
		if targetPeriod == nil {
			return errNotFound
		}
		if (currentPeriod != nil && currentPeriod.LockedAt != nil) || targetPeriod.LockedAt != nil {
			return errConflict
		}
		candidate := append([]domain.Transaction{}, db.Transactions...)
		candidate[index] = item
		if issues := validation.Transactions(candidate); len(issues) > 0 {
			return validationError{issues}
		}
		db.Transactions[index] = item
		prependAudit(db, s.now(), "Cập nhật giao dịch", fmt.Sprintf("%s · ngày %s · %d VND", item.Description, item.Date, item.Amount))
		return nil
	})
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "transaction_or_period_not_found", "không tìm thấy giao dịch hoặc kỳ kê khai")
		return
	}
	if errors.Is(err, errConflict) {
		writeError(w, http.StatusConflict, "period_locked", "kỳ kê khai đã khóa")
		return
	}
	var ve validationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "validation_error", "message": "giao dịch không hợp lệ", "issues": ve.issues}})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) calculate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PeriodID string `json:"periodId"`
	}
	if err := decode(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	db, err := s.repo.Snapshot(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	period := findPeriod(db.Periods, input.PeriodID)
	if period == nil {
		writeError(w, http.StatusNotFound, "period_not_found", "không tìm thấy kỳ kê khai")
		return
	}
	if period.TaxSnapshot != nil {
		writeJSON(w, http.StatusOK, period.TaxSnapshot)
		return
	}
	result, err := taxengine.Calculate(*period, db.Transactions, s.now())
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "calculation_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) importTransactions(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Items []domain.Transaction `json:"items"`
	}
	if err := decode(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if len(input.Items) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "empty_import", "file import không có dữ liệu")
		return
	}
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		for i := range input.Items {
			if input.Items[i].ID == "" {
				input.Items[i].ID = newID("tx")
			}
			period := findPeriod(db.Periods, input.Items[i].PeriodID)
			if period == nil {
				return errNotFound
			}
			if period.LockedAt != nil {
				return errConflict
			}
		}
		candidate := append(append([]domain.Transaction{}, db.Transactions...), input.Items...)
		if issues := validation.Transactions(candidate); len(issues) > 0 {
			return validationError{issues}
		}
		db.Transactions = append(input.Items, db.Transactions...)
		prependAudit(db, s.now(), "Import dữ liệu", fmt.Sprintf("Đã nhập %d giao dịch", len(input.Items)))
		return nil
	})
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "period_not_found", "import chứa kỳ kê khai không tồn tại")
		return
	}
	if errors.Is(err, errConflict) {
		writeError(w, http.StatusConflict, "period_locked", "không thể import vào kỳ đã khóa")
		return
	}
	var ve validationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "validation_error", "message": "dữ liệu import không hợp lệ", "issues": ve.issues}})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": len(input.Items)})
}

func (s *Server) exportData(w http.ResponseWriter, r *http.Request) {
	periodID := r.URL.Query().Get("periodId")
	if periodID == "" {
		writeError(w, http.StatusBadRequest, "period_required", "thiếu periodId")
		return
	}
	db, err := s.repo.Snapshot(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	period := findPeriod(db.Periods, periodID)
	if period == nil {
		writeError(w, http.StatusNotFound, "period_not_found", "không tìm thấy kỳ kê khai")
		return
	}
	result := period.TaxSnapshot
	if result == nil {
		calculated, err := taxengine.Calculate(*period, db.Transactions, s.now())
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "calculation_error", err.Error())
			return
		}
		result = &calculated
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=tax-export-%s.json", periodID))
	writeJSON(w, http.StatusOK, map[string]any{"profile": db.Profile, "period": period, "result": result, "transactions": filterTransactions(db.Transactions, periodID)})
}

func (s *Server) declarations(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		db, err := s.repo.Snapshot(r.Context())
		if err != nil {
			s.internal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, db.Declarations)
		return
	}
	var input domain.TaxDeclaration
	if err := decode(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if input.ID != r.PathValue("id") {
		writeError(w, http.StatusBadRequest, "id_mismatch", "id trong URL và payload không khớp")
		return
	}
	if err := validateDeclaration(input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	err := s.repo.Update(r.Context(), func(db *domain.Database) error {
		for i := range db.Declarations {
			if db.Declarations[i].ID != input.ID {
				continue
			}
			if db.Declarations[i].LockedAt != nil {
				return errConflict
			}
			input.UpdatedAt = s.now().UTC()
			db.Declarations[i] = input
			prependAudit(db, s.now(), "Cập nhật tờ khai doanh nghiệp", input.FormCode+" · "+input.PeriodLabel)
			return nil
		}
		return errNotFound
	})
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "declaration_not_found", "không tìm thấy tờ khai")
		return
	}
	if errors.Is(err, errConflict) {
		writeError(w, http.StatusConflict, "declaration_locked", "tờ khai đã khóa")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, input)
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	db, err := s.repo.Snapshot(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, db.Audit)
}

type validationError struct{ issues []validation.Issue }

func (e validationError) Error() string { return "validation failed" }

func validateProfile(p domain.BusinessProfile) error {
	if p.TaxpayerType != "household" && p.TaxpayerType != "enterprise" {
		return errors.New("taxpayerType phải là household hoặc enterprise")
	}
	if strings.TrimSpace(p.TaxCode) == "" || strings.TrimSpace(p.BusinessName) == "" {
		return errors.New("thiếu mã số thuế hoặc tên người nộp thuế")
	}
	if p.DeclarationKind != "month" && p.DeclarationKind != "quarter" {
		return errors.New("declarationKind không hợp lệ")
	}
	if p.TaxpayerType == "enterprise" && strings.TrimSpace(p.LegalRepresentative) == "" {
		return errors.New("doanh nghiệp phải có người đại diện pháp luật")
	}
	if p.TaxpayerType == "household" && p.HouseholdTaxMethod != "" && p.HouseholdTaxMethod != "non_taxable" && p.HouseholdTaxMethod != "revenue_percentage" && p.HouseholdTaxMethod != "taxable_income" {
		return errors.New("householdTaxMethod không hợp lệ")
	}
	return nil
}

func validatePeriod(p domain.TaxPeriod) error {
	if p.Label == "" || p.Year < 2000 || (p.Kind != "month" && p.Kind != "quarter") {
		return errors.New("thông tin kỳ kê khai không hợp lệ")
	}
	if _, err := time.Parse(time.DateOnly, p.DueDate); err != nil {
		return errors.New("dueDate phải có dạng YYYY-MM-DD")
	}
	if p.PaidAmount < 0 {
		return errors.New("paidAmount không được âm")
	}
	return nil
}

func validateDeclaration(d domain.TaxDeclaration) error {
	if d.ID == "" || d.FormCode == "" || d.SchemaVersion == "" {
		return errors.New("thiếu mã tờ khai hoặc phiên bản schema")
	}
	if d.TaxType != "vat" && d.TaxType != "pit" && d.TaxType != "cit" {
		return errors.New("taxType không hợp lệ")
	}
	for key, value := range d.Values {
		if value < 0 {
			return fmt.Errorf("chỉ tiêu %s không được âm", key)
		}
	}
	return nil
}

func validateCounterparty(item domain.Counterparty) error {
	if code := strings.TrimSpace(item.Code); code == "" || len(code) > 64 {
		return errors.New("mã đối tượng phải có từ 1 đến 64 ký tự")
	}
	if name := strings.TrimSpace(item.Name); name == "" || len(name) > 255 {
		return errors.New("tên đối tượng phải có từ 1 đến 255 ký tự")
	}
	if len(strings.TrimSpace(item.TaxCode)) > 32 {
		return errors.New("mã số thuế không được quá 32 ký tự")
	}
	if len(strings.TrimSpace(item.Address)) > 500 {
		return errors.New("địa chỉ không được quá 500 ký tự")
	}
	return nil
}

func normalizeCashReceipt(voucherID string, input *domain.CashReceipt) {
	if len(input.Invoices) == 0 && strings.TrimSpace(input.InvoiceNo) != "" {
		input.Invoices = []domain.ReceiptInvoice{{ID: newID("inv"), InvoiceNo: strings.TrimSpace(input.InvoiceNo), Symbol: strings.TrimSpace(input.InvoiceSymbol), InvoiceDate: input.InvoiceDate, TaxCode: strings.TrimSpace(input.CounterpartyTaxCode)}}
	}
	for index := range input.Invoices {
		invoice := &input.Invoices[index]
		if invoice.ID == "" {
			invoice.ID = newID("inv")
		}
		invoice.VoucherID = voucherID
	}
	defaultInvoiceID := ""
	if len(input.Invoices) == 1 {
		defaultInvoiceID = input.Invoices[0].ID
	}
	revenueByInvoice := map[string]string{}
	for index := range input.Entries {
		entry := &input.Entries[index]
		if entry.ID == "" {
			entry.ID = newID("detail")
		}
		entry.VoucherID = voucherID
		if entry.InvoiceID == "" {
			entry.InvoiceID = defaultInvoiceID
		}
		if entry.Kind == "normal" {
			if entry.DetailCode == "" {
				entry.DetailCode = strings.TrimSpace(input.DetailCode)
			}
			if entry.DetailCode == "" {
				entry.DetailCode = "DOANH_THU"
			}
			if entry.Quantity == "" {
				entry.Quantity = input.Quantity
			}
			if entry.Quantity == "" {
				entry.Quantity = "1"
			}
			if entry.UnitPrice == 0 {
				entry.UnitPrice = input.UnitPrice
			}
			if entry.UnitPrice == 0 {
				entry.UnitPrice = entry.Amount
			}
			revenueByInvoice[entry.InvoiceID] = entry.ID
		}
	}
	if len(input.TaxLines) == 0 {
		for index := range input.Entries {
			entry := &input.Entries[index]
			if entry.Kind != "vat" {
				continue
			}
			if entry.RevenueDetailID == "" {
				entry.RevenueDetailID = revenueByInvoice[entry.InvoiceID]
			}
			taxableAmount := domain.Money(0)
			for _, detail := range input.Entries {
				if detail.ID == entry.RevenueDetailID {
					taxableAmount = detail.Amount
					break
				}
			}
			input.TaxLines = append(input.TaxLines, domain.ReceiptTaxLine{ID: newID("vat"), VoucherID: voucherID, InvoiceID: entry.InvoiceID, RevenueDetailID: entry.RevenueDetailID, TaxRate: entry.Rate, TaxableAmount: taxableAmount, TaxAmount: entry.Amount, PriceIncludesTax: input.AmountIncludesVAT})
		}
	}
	for index := range input.TaxLines {
		input.TaxLines[index].VoucherID = voucherID
	}
}

func validateCashReceipt(input domain.CashReceipt) error {
	if input.Status != "draft" && input.Status != "saved" {
		return errors.New("trạng thái phiếu thu không hợp lệ")
	}
	if strings.TrimSpace(input.PeriodID) == "" || strings.TrimSpace(input.ReceiptNo) == "" || strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.CounterpartyName) == "" {
		return errors.New("thiếu kỳ kê khai, số phiếu thu, người nộp hoặc lý do thu")
	}
	if _, err := time.Parse(time.DateOnly, input.VoucherDate); err != nil {
		return errors.New("ngày chứng từ phải có dạng YYYY-MM-DD")
	}
	if _, err := time.Parse(time.DateOnly, input.AccountingDate); err != nil {
		return errors.New("ngày hạch toán phải có dạng YYYY-MM-DD")
	}
	if !validAccount(input.DebitAccount) || !validAccount(input.CreditAccount) {
		return errors.New("tài khoản Nợ/Có phải gồm 3 đến 10 chữ số")
	}
	invoices := make(map[string]domain.ReceiptInvoice, len(input.Invoices))
	for _, invoice := range input.Invoices {
		if invoice.ID == "" || invoice.VoucherID == "" || strings.TrimSpace(invoice.InvoiceNo) == "" {
			return errors.New("hóa đơn phải có ID, voucher_id và số hóa đơn")
		}
		if _, duplicated := invoices[invoice.ID]; duplicated {
			return errors.New("ID hóa đơn bị trùng")
		}
		if invoice.InvoiceDate != "" {
			if _, err := time.Parse(time.DateOnly, invoice.InvoiceDate); err != nil {
				return errors.New("ngày hóa đơn phải có dạng YYYY-MM-DD")
			}
		}
		invoices[invoice.ID] = invoice
	}
	revenueDetails := map[string]domain.ReceiptAccountingEntry{}
	vatEntries := map[string]domain.ReceiptAccountingEntry{}
	var debitTotal, creditTotal, receiptTotal domain.Money
	for _, entry := range input.Entries {
		if !validAccount(entry.DebitAccount) || !validAccount(entry.CreditAccount) || entry.Amount < 0 {
			return errors.New("dòng định khoản phải có tài khoản Nợ/Có và số tiền hợp lệ")
		}
		if entry.Kind != "normal" && entry.Kind != "vat" && entry.Kind != "cogs" {
			return errors.New("loại dòng định khoản không hợp lệ")
		}
		if entry.Kind == "vat" && (entry.CreditAccount != "33311" || !validVATRate(entry.Rate)) {
			return errors.New("định khoản thuế GTGT hoặc tỷ lệ thuế không hợp lệ")
		}
		if entry.Kind == "normal" {
			if entry.ID == "" || entry.VoucherID == "" || entry.InvoiceID == "" || strings.TrimSpace(entry.DetailCode) == "" || entry.UnitPrice < 0 || entry.Amount <= 0 {
				return errors.New("dòng doanh thu thiếu ID liên kết, mã hàng, số lượng hoặc đơn giá")
			}
			if _, found := invoices[entry.InvoiceID]; !found {
				return errors.New("dòng doanh thu không liên kết đúng hóa đơn")
			}
			expectedAmount, err := quantityTimesMoney(entry.Quantity, entry.UnitPrice)
			if err != nil || expectedAmount != entry.Amount {
				return errors.New("số tiền dòng doanh thu phải bằng số lượng nhân đơn giá và làm tròn đến đồng")
			}
			revenueDetails[entry.ID] = entry
			receiptTotal += entry.Amount
		}
		if entry.Kind == "vat" {
			if entry.ID == "" || entry.InvoiceID == "" || entry.RevenueDetailID == "" {
				return errors.New("dòng thuế thiếu ID hóa đơn hoặc revenue_detail_id")
			}
			vatEntries[entry.RevenueDetailID] = entry
			receiptTotal += entry.Amount
		}
		if entry.Kind == "cogs" && entry.Amount <= 0 {
			return errors.New("số tiền dòng giá vốn phải lớn hơn 0")
		}
		debitTotal += entry.Amount
		creditTotal += entry.Amount
	}
	if input.Amount <= 0 || input.ConvertedAmount <= 0 {
		return errors.New("số tiền và số tiền quy đổi phải là số nguyên VND dương")
	}
	if strings.TrimSpace(input.Currency) == "" || input.ExchangeRate <= 0 {
		return errors.New("loại tiền hoặc tỷ giá không hợp lệ")
	}
	for _, taxLine := range input.TaxLines {
		detail, found := revenueDetails[taxLine.RevenueDetailID]
		if taxLine.ID == "" || taxLine.VoucherID == "" || !found || detail.InvoiceID != taxLine.InvoiceID || !validVATRate(taxLine.TaxRate) {
			return errors.New("dòng thuế không liên kết đúng phiếu thu, hóa đơn và dòng doanh thu")
		}
		if taxLine.TaxableAmount != detail.Amount {
			return errors.New("giá tính thuế không khớp dòng doanh thu được liên kết")
		}
		expectedTax := domain.Money(math.Round(float64(taxLine.TaxableAmount) * taxLine.TaxRate / 100))
		entry, found := vatEntries[taxLine.RevenueDetailID]
		if taxLine.TaxAmount != expectedTax || !found || entry.Amount != expectedTax || entry.InvoiceID != taxLine.InvoiceID {
			return errors.New("tiền thuế GTGT không đúng hoặc không khớp dòng định khoản 33311")
		}
	}
	if len(input.TaxLines) != len(vatEntries) {
		return errors.New("mỗi dòng thuế phải liên kết duy nhất với một dòng doanh thu")
	}
	if debitTotal != creditTotal {
		return fmt.Errorf("định khoản không cân bằng, chênh lệch %d đồng", debitTotal-creditTotal)
	}
	if len(input.Entries) > 0 {
		if input.ConvertedAmount != domain.Money(math.Round(float64(receiptTotal)*input.ExchangeRate)) {
			return errors.New("số tiền quy đổi không khớp tổng tiền Nợ của phiếu thu")
		}
	}
	if input.InvoiceDate != "" {
		if _, err := time.Parse(time.DateOnly, input.InvoiceDate); err != nil {
			return errors.New("ngày hóa đơn phải có dạng YYYY-MM-DD")
		}
	}
	if len(input.Attachments) > 5 {
		return errors.New("chỉ được đính kèm tối đa 5 tệp")
	}
	var attachmentBytes int64
	for _, attachment := range input.Attachments {
		if strings.TrimSpace(attachment.Name) == "" || attachment.Size < 0 || attachment.Size > 5<<20 {
			return errors.New("tệp đính kèm không hợp lệ hoặc vượt quá 5 MB")
		}
		attachmentBytes += attachment.Size
	}
	if attachmentBytes > 5<<20 {
		return errors.New("tổng dung lượng tệp đính kèm vượt quá 5 MB")
	}
	if input.Status == "saved" && strings.TrimSpace(input.CounterpartyCode) == "" {
		return errors.New("thiếu mã đối tượng")
	}
	if input.Status == "draft" && strings.TrimSpace(input.CounterpartyCode) == "" {
		input.CounterpartyCode = "DRAFT"
	}
	if input.Status == "draft" && input.RevenueCategory == "" {
		input.RevenueCategory = "other"
	}
	if input.Status == "draft" && input.Amount <= 0 {
		return errors.New("số tiền thu phải lớn hơn 0")
	}
	if input.Status == "draft" && input.ConvertedAmount <= 0 {
		return errors.New("số tiền quy đổi phải lớn hơn 0")
	}
	if input.RevenueCategory != "distribution" && input.RevenueCategory != "services" && input.RevenueCategory != "production" && input.RevenueCategory != "other" {
		return errors.New("nhóm ngành không hợp lệ")
	}
	if err := validateCounterparty(domain.Counterparty{Code: input.CounterpartyCode, Name: input.CounterpartyName, TaxCode: input.CounterpartyTaxCode, Address: input.CounterpartyAddress}); err != nil {
		return err
	}
	return nil
}

func validAccount(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 3 || len(value) > 10 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func validVATRate(rate float64) bool {
	switch rate {
	case 0, 5, 8, 10:
		return true
	default:
		return false
	}
}

func quantityTimesMoney(quantity string, unitPrice domain.Money) (domain.Money, error) {
	quantity = strings.TrimSpace(quantity)
	if quantity == "" || unitPrice < 0 {
		return 0, errors.New("số lượng hoặc đơn giá không hợp lệ")
	}
	decimalPointSeen := false
	for _, char := range quantity {
		switch {
		case char >= '0' && char <= '9':
		case char == '.' && !decimalPointSeen:
			decimalPointSeen = true
		default:
			return 0, errors.New("số lượng phải là số thập phân dương")
		}
	}

	parsed, ok := new(big.Rat).SetString(quantity)
	if !ok || parsed.Sign() <= 0 {
		return 0, errors.New("số lượng phải lớn hơn 0")
	}
	product := new(big.Rat).Mul(parsed, new(big.Rat).SetInt64(int64(unitPrice)))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(product.Num(), product.Denom(), remainder)
	if new(big.Int).Lsh(remainder, 1).Cmp(product.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() {
		return 0, errors.New("số tiền vượt quá giới hạn")
	}
	return domain.Money(quotient.Int64()), nil
}

func receiptTransaction(id string, input domain.CashReceipt) domain.Transaction {
	var vatAmount domain.Money
	for _, entry := range input.Entries {
		if entry.Kind == "vat" {
			vatAmount += entry.Amount
		}
	}
	return domain.Transaction{ID: id, PeriodID: input.PeriodID, Date: input.VoucherDate, Type: "revenue", Description: strings.TrimSpace(input.Description), InvoiceNo: strings.TrimSpace(input.ReceiptNo), DocumentNo: strings.TrimSpace(input.ReceiptNo), Amount: input.ConvertedAmount, VATAmount: vatAmount, RevenueCategory: input.RevenueCategory, PaymentStatus: "paid", VoucherType: "cash_receipt", CounterpartyCode: strings.TrimSpace(input.CounterpartyCode), CounterpartyName: strings.TrimSpace(input.CounterpartyName), CounterpartyTaxCode: strings.TrimSpace(input.CounterpartyTaxCode), CounterpartyAddress: strings.TrimSpace(input.CounterpartyAddress), CashReceipt: &domain.CashReceiptData{VoucherDate: input.VoucherDate, AccountingDate: input.AccountingDate, Status: input.Status, ContactName: input.ContactName, DebitAccount: input.DebitAccount, CreditAccount: input.CreditAccount, Currency: input.Currency, ExchangeRate: input.ExchangeRate, ConvertedAmount: input.ConvertedAmount, AmountIncludesVAT: input.AmountIncludesVAT, InvoiceNo: input.InvoiceNo, InvoiceSymbol: input.InvoiceSymbol, InvoiceDate: input.InvoiceDate, DetailCode: input.DetailCode, Quantity: input.Quantity, UnitPrice: input.UnitPrice, CaseCode: input.CaseCode, Collector: input.Collector, Note: input.Note, Attachments: input.Attachments, Entries: input.Entries, Invoices: input.Invoices, TaxLines: input.TaxLines}}
}

func findPeriod(items []domain.TaxPeriod, id string) *domain.TaxPeriod {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

func upsertCounterparty(items *[]domain.Counterparty, input domain.Counterparty) {
	for i := range *items {
		if (*items)[i].Code == input.Code {
			(*items)[i] = input
			return
		}
	}
	*items = append(*items, input)
}

func hasActiveAccount(items []domain.Account, code string) bool {
	for _, item := range items {
		if item.Code == code && item.IsActive {
			return true
		}
	}
	return false
}

func validateReceiptAccountCatalog(accounts []domain.Account, receipt *domain.CashReceiptData) error {
	if receipt == nil || !hasActiveAccount(accounts, receipt.DebitAccount) || !hasActiveAccount(accounts, receipt.CreditAccount) {
		return errors.New("tài khoản Nợ/Có không tồn tại hoặc đã ngừng sử dụng")
	}
	for _, entry := range receipt.Entries {
		if !hasActiveAccount(accounts, entry.DebitAccount) || !hasActiveAccount(accounts, entry.CreditAccount) {
			return fmt.Errorf("tài khoản định khoản %s/%s không tồn tại hoặc đã ngừng sử dụng", entry.DebitAccount, entry.CreditAccount)
		}
	}
	return nil
}
func filterTransactions(items []domain.Transaction, periodID string) []domain.Transaction {
	out := make([]domain.Transaction, 0)
	for _, item := range items {
		if item.PeriodID == periodID {
			out = append(out, item)
		}
	}
	return out
}
func prependAudit(db *domain.Database, at time.Time, action, detail string) {
	db.Audit = append([]domain.AuditEntry{{ID: newID("audit"), At: at.UTC(), Action: action, Detail: detail}}, db.Audit...)
}

func newID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return prefix + "-" + hex.EncodeToString(b)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != nil {
		return nil
	}
	return errors.New("request chỉ được chứa một JSON object")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func (s *Server) internal(w http.ResponseWriter, err error) {
	s.logger.Error("request failed", "error", err)
	writeError(w, http.StatusInternalServerError, "internal_error", "lỗi hệ thống")
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "Bearer " + s.token
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(expected)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized", "thiếu hoặc sai access token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == s.allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", s.allowedOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
	})
}
func (s *Server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic recovered", "panic", recovered)
				writeError(w, http.StatusInternalServerError, "internal_error", "lỗi hệ thống")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
