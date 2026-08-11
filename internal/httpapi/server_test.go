package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"tax-client/backend/internal/repository"
	"tax-client/backend/internal/seed"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	repo, err := repository.NewJSONRepository(filepath.Join(t.TempDir(), "tax.json"), seed.Database())
	if err != nil {
		t.Fatal(err)
	}
	return New(repo, Config{Token: "test-token", Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: func() time.Time { return time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC) }})
}

func request(t *testing.T, handler http.Handler, method, path string, body any, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if authenticated {
		req.Header.Set("Authorization", "Bearer test-token")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func TestLoginAndAuthorization(t *testing.T) {
	handler := newTestServer(t)
	unauthorized := request(t, handler, http.MethodGet, "/api/tax-periods", nil, false)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", unauthorized.Code)
	}
	login := request(t, handler, http.MethodPost, "/api/login", map[string]string{"username": "demo", "password": "demo123"}, false)
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", login.Code, login.Body.String())
	}
}

func TestCalculateThenLockPeriod(t *testing.T) {
	handler := newTestServer(t)
	transaction := map[string]any{"periodId": "2026-q3", "date": "2026-08-11", "type": "revenue", "description": "Sale", "invoiceNo": "HD-TEST", "amount": 10000, "revenueCategory": "distribution"}
	created := request(t, handler, http.MethodPost, "/api/transactions", transaction, true)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", created.Code, created.Body.String())
	}
	calculated := request(t, handler, http.MethodPost, "/api/calculate", map[string]string{"periodId": "2026-q3"}, true)
	if calculated.Code != http.StatusOK {
		t.Fatalf("calculate status = %d body=%s", calculated.Code, calculated.Body.String())
	}
	var result struct {
		TotalTax       int64  `json:"totalTax"`
		FormulaVersion string `json:"formulaVersion"`
	}
	if err := json.Unmarshal(calculated.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.TotalTax != 150 || result.FormulaVersion == "" {
		t.Fatalf("unexpected result: %+v", result)
	}

	locked := request(t, handler, http.MethodPost, "/api/tax-periods/2026-q3/lock", nil, true)
	if locked.Code != http.StatusOK {
		t.Fatalf("lock status = %d body=%s", locked.Code, locked.Body.String())
	}
	lateTransaction := map[string]any{"periodId": "2026-q3", "date": "2026-08-12", "type": "revenue", "description": "Late", "invoiceNo": "HD-LATE", "amount": 1000, "revenueCategory": "distribution"}
	late := request(t, handler, http.MethodPost, "/api/transactions", lateTransaction, true)
	if late.Code != http.StatusConflict {
		t.Fatalf("create in locked period status = %d body=%s", late.Code, late.Body.String())
	}
}

func TestImportRejectsDuplicate(t *testing.T) {
	handler := newTestServer(t)
	item := map[string]any{"id": "duplicate-a", "periodId": "2026-q3", "date": "2026-07-08", "type": "revenue", "description": "Duplicate", "invoiceNo": "hd-test", "amount": 1000, "revenueCategory": "distribution"}
	duplicate := map[string]any{"id": "duplicate-b", "periodId": "2026-q3", "date": "2026-07-08", "type": "revenue", "description": "Duplicate", "invoiceNo": " HD-TEST ", "amount": 1000, "revenueCategory": "distribution"}
	response := request(t, handler, http.MethodPost, "/api/imports", map[string]any{"items": []any{item, duplicate}}, true)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestDeleteTransaction(t *testing.T) {
	handler := newTestServer(t)
	transaction := map[string]any{"id": "to-delete", "periodId": "2026-q3", "date": "2026-08-11", "type": "expense", "description": "Wrong", "invoiceNo": "CT-WRONG", "amount": 1000, "revenueCategory": "distribution"}
	created := request(t, handler, http.MethodPost, "/api/transactions", transaction, true)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", created.Code, created.Body.String())
	}
	deleted := request(t, handler, http.MethodDelete, "/api/transactions/to-delete", nil, true)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body=%s", deleted.Code, deleted.Body.String())
	}
	listed := request(t, handler, http.MethodGet, "/api/transactions?periodId=2026-q3", nil, true)
	var items []map[string]any
	if err := json.Unmarshal(listed.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items after delete = %d, want 0", len(items))
	}
}
