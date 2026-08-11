package repository

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tax-client/backend/internal/domain"
)

//go:embed migrations/*.sql
var migrations embed.FS

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(ctx context.Context, dsn string, seed domain.Database) (*PostgresRepository, error) {
	pool, err := newPostgresPool(ctx, dsn)
	if err != nil {
		return nil, err
	}
	r := &PostgresRepository{pool: pool}
	if err := r.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	var exists bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM business_profiles WHERE id = 1)").Scan(&exists); err != nil {
		pool.Close()
		return nil, err
	}
	if !exists {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			pool.Close()
			return nil, err
		}
		defer tx.Rollback(ctx)
		if err := syncDatabase(ctx, tx, seed); err != nil {
			pool.Close()
			return nil, fmt.Errorf("seed postgres: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("commit seed: %w", err)
		}
	}
	return r, nil
}

func RunPostgresMigrations(ctx context.Context, dsn string) error {
	pool, err := newPostgresPool(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	return (&PostgresRepository{pool: pool}).migrate(ctx)
}

func newPostgresPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	config.MaxConns = 10
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func (r *PostgresRepository) Close() { r.pool.Close() }

func (r *PostgresRepository) migrate(ctx context.Context) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(721527)); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("create migration registry: %w", err)
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		var applied bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name=$1)", entry.Name()).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied {
			continue
		}
		sql, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("run migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations(name) VALUES($1)", entry.Name()); err != nil {
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) Snapshot(ctx context.Context) (domain.Database, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return domain.Database{}, err
	}
	defer tx.Rollback(ctx)
	db, err := loadDatabase(ctx, tx)
	if err != nil {
		return domain.Database{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Database{}, err
	}
	return db, nil
}

func (r *PostgresRepository) Update(ctx context.Context, mutate func(*domain.Database) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Current API mutates an aggregate. Serialize writes until repositories expose granular commands.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(721526)); err != nil {
		return err
	}
	db, err := loadDatabase(ctx, tx)
	if err != nil {
		return err
	}
	if err := mutate(&db); err != nil {
		return err
	}
	if err := syncDatabase(ctx, tx, db); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadDatabase(ctx context.Context, q querier) (domain.Database, error) {
	var db domain.Database
	err := q.QueryRow(ctx, `SELECT taxpayer_type, tax_code, business_name, owner_name, address, industry, declaration_kind,
		COALESCE(enterprise_type,''), COALESCE(legal_representative,''), COALESCE(vat_method,''), COALESCE(accounting_regime,''), COALESCE(household_tax_method,'')
		FROM business_profiles WHERE id = 1`).Scan(&db.Profile.TaxpayerType, &db.Profile.TaxCode, &db.Profile.BusinessName, &db.Profile.OwnerName, &db.Profile.Address, &db.Profile.Industry, &db.Profile.DeclarationKind, &db.Profile.EnterpriseType, &db.Profile.LegalRepresentative, &db.Profile.VATMethod, &db.Profile.AccountingRegime, &db.Profile.HouseholdTaxMethod)
	if err != nil {
		return db, fmt.Errorf("load profile: %w", err)
	}

	rows, err := q.Query(ctx, `SELECT id, label, year, kind, status, due_date, paid_amount, locked_at, tax_snapshot FROM tax_periods ORDER BY year DESC, due_date DESC`)
	if err != nil {
		return db, err
	}
	for rows.Next() {
		var item domain.TaxPeriod
		var due time.Time
		var snapshot []byte
		if err := rows.Scan(&item.ID, &item.Label, &item.Year, &item.Kind, &item.Status, &due, &item.PaidAmount, &item.LockedAt, &snapshot); err != nil {
			rows.Close()
			return db, err
		}
		item.DueDate = due.Format(time.DateOnly)
		if len(snapshot) > 0 {
			var value domain.TaxBreakdown
			if err := json.Unmarshal(snapshot, &value); err != nil {
				rows.Close()
				return db, err
			}
			item.TaxSnapshot = &value
		}
		db.Periods = append(db.Periods, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return db, err
	}
	rows.Close()

	rows, err = q.Query(ctx, `SELECT id, period_id, transaction_date, type, description, invoice_no, amount, revenue_category,
		COALESCE(document_no,''), COALESCE(payment_status,''), outstanding_amount FROM transactions ORDER BY transaction_date DESC, created_at DESC`)
	if err != nil {
		return db, err
	}
	for rows.Next() {
		var item domain.Transaction
		var date time.Time
		if err := rows.Scan(&item.ID, &item.PeriodID, &date, &item.Type, &item.Description, &item.InvoiceNo, &item.Amount, &item.RevenueCategory, &item.DocumentNo, &item.PaymentStatus, &item.OutstandingAmount); err != nil {
			rows.Close()
			return db, err
		}
		item.Date = date.Format(time.DateOnly)
		db.Transactions = append(db.Transactions, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return db, err
	}
	rows.Close()

	rows, err = q.Query(ctx, `SELECT id, tax_type, form_code, form_name, period_label, period_type, status, schema_version, values, updated_at, locked_at FROM tax_declarations ORDER BY updated_at DESC`)
	if err != nil {
		return db, err
	}
	for rows.Next() {
		var item domain.TaxDeclaration
		var values []byte
		if err := rows.Scan(&item.ID, &item.TaxType, &item.FormCode, &item.FormName, &item.PeriodLabel, &item.PeriodType, &item.Status, &item.SchemaVersion, &values, &item.UpdatedAt, &item.LockedAt); err != nil {
			rows.Close()
			return db, err
		}
		if err := json.Unmarshal(values, &item.Values); err != nil {
			rows.Close()
			return db, err
		}
		db.Declarations = append(db.Declarations, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return db, err
	}
	rows.Close()

	rows, err = q.Query(ctx, `SELECT id, occurred_at, action, detail FROM audit_entries ORDER BY occurred_at DESC`)
	if err != nil {
		return db, err
	}
	for rows.Next() {
		var item domain.AuditEntry
		if err := rows.Scan(&item.ID, &item.At, &item.Action, &item.Detail); err != nil {
			rows.Close()
			return db, err
		}
		db.Audit = append(db.Audit, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return db, err
	}
	rows.Close()
	return db, nil
}

func syncDatabase(ctx context.Context, tx pgx.Tx, db domain.Database) error {
	p := db.Profile
	_, err := tx.Exec(ctx, `INSERT INTO business_profiles (id, taxpayer_type, tax_code, business_name, owner_name, address, industry, declaration_kind, enterprise_type, legal_representative, vat_method, accounting_regime, household_tax_method)
		VALUES (1,$1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),NULLIF($9,''),NULLIF($10,''),NULLIF($11,''),NULLIF($12,''))
		ON CONFLICT (id) DO UPDATE SET taxpayer_type=EXCLUDED.taxpayer_type,tax_code=EXCLUDED.tax_code,business_name=EXCLUDED.business_name,owner_name=EXCLUDED.owner_name,address=EXCLUDED.address,industry=EXCLUDED.industry,declaration_kind=EXCLUDED.declaration_kind,enterprise_type=EXCLUDED.enterprise_type,legal_representative=EXCLUDED.legal_representative,vat_method=EXCLUDED.vat_method,accounting_regime=EXCLUDED.accounting_regime,household_tax_method=EXCLUDED.household_tax_method,updated_at=NOW()`, p.TaxpayerType, p.TaxCode, p.BusinessName, p.OwnerName, p.Address, p.Industry, p.DeclarationKind, p.EnterpriseType, p.LegalRepresentative, p.VATMethod, p.AccountingRegime, p.HouseholdTaxMethod)
	if err != nil {
		return fmt.Errorf("save profile: %w", err)
	}

	periodIDs := make([]string, 0, len(db.Periods))
	for _, item := range db.Periods {
		periodIDs = append(periodIDs, item.ID)
		var snapshot any
		if item.TaxSnapshot != nil {
			raw, err := json.Marshal(item.TaxSnapshot)
			if err != nil {
				return err
			}
			snapshot = string(raw)
		}
		_, err := tx.Exec(ctx, `INSERT INTO tax_periods (id,label,year,kind,status,due_date,paid_amount,locked_at,tax_snapshot) VALUES ($1,$2,$3,$4,$5,$6::date,$7,$8,$9::jsonb)
			ON CONFLICT (id) DO UPDATE SET label=EXCLUDED.label,year=EXCLUDED.year,kind=EXCLUDED.kind,status=EXCLUDED.status,due_date=EXCLUDED.due_date,paid_amount=EXCLUDED.paid_amount,locked_at=EXCLUDED.locked_at,tax_snapshot=EXCLUDED.tax_snapshot`, item.ID, item.Label, item.Year, item.Kind, item.Status, item.DueDate, item.PaidAmount, item.LockedAt, snapshot)
		if err != nil {
			return fmt.Errorf("save period %s: %w", item.ID, err)
		}
	}
	transactionIDs := make([]string, 0, len(db.Transactions))
	for _, item := range db.Transactions {
		transactionIDs = append(transactionIDs, item.ID)
		_, err := tx.Exec(ctx, `INSERT INTO transactions (id,period_id,transaction_date,type,description,invoice_no,amount,revenue_category,document_no,payment_status,outstanding_amount) VALUES ($1,$2,$3::date,$4,$5,$6,$7,$8,NULLIF($9,''),NULLIF($10,''),$11)
			ON CONFLICT (id) DO UPDATE SET period_id=EXCLUDED.period_id,transaction_date=EXCLUDED.transaction_date,type=EXCLUDED.type,description=EXCLUDED.description,invoice_no=EXCLUDED.invoice_no,amount=EXCLUDED.amount,revenue_category=EXCLUDED.revenue_category,document_no=EXCLUDED.document_no,payment_status=EXCLUDED.payment_status,outstanding_amount=EXCLUDED.outstanding_amount,updated_at=NOW()`, item.ID, item.PeriodID, item.Date, item.Type, item.Description, item.InvoiceNo, item.Amount, item.RevenueCategory, item.DocumentNo, item.PaymentStatus, item.OutstandingAmount)
		if err != nil {
			return fmt.Errorf("save transaction %s: %w", item.ID, err)
		}
	}
	if err := deleteMissing(ctx, tx, "transactions", transactionIDs); err != nil {
		return err
	}
	if err := deleteMissing(ctx, tx, "tax_periods", periodIDs); err != nil {
		return err
	}

	declarationIDs := make([]string, 0, len(db.Declarations))
	for _, item := range db.Declarations {
		declarationIDs = append(declarationIDs, item.ID)
		values, err := json.Marshal(item.Values)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO tax_declarations (id,tax_type,form_code,form_name,period_label,period_type,status,schema_version,values,updated_at,locked_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11)
			ON CONFLICT (id) DO UPDATE SET tax_type=EXCLUDED.tax_type,form_code=EXCLUDED.form_code,form_name=EXCLUDED.form_name,period_label=EXCLUDED.period_label,period_type=EXCLUDED.period_type,status=EXCLUDED.status,schema_version=EXCLUDED.schema_version,values=EXCLUDED.values,updated_at=EXCLUDED.updated_at,locked_at=EXCLUDED.locked_at`, item.ID, item.TaxType, item.FormCode, item.FormName, item.PeriodLabel, item.PeriodType, item.Status, item.SchemaVersion, string(values), item.UpdatedAt, item.LockedAt)
		if err != nil {
			return fmt.Errorf("save declaration %s: %w", item.ID, err)
		}
	}
	if err := deleteMissing(ctx, tx, "tax_declarations", declarationIDs); err != nil {
		return err
	}

	auditIDs := make([]string, 0, len(db.Audit))
	for _, item := range db.Audit {
		auditIDs = append(auditIDs, item.ID)
		if _, err := tx.Exec(ctx, `INSERT INTO audit_entries (id,occurred_at,action,detail) VALUES ($1,$2,$3,$4) ON CONFLICT (id) DO UPDATE SET occurred_at=EXCLUDED.occurred_at,action=EXCLUDED.action,detail=EXCLUDED.detail`, item.ID, item.At, item.Action, item.Detail); err != nil {
			return fmt.Errorf("save audit %s: %w", item.ID, err)
		}
	}
	return deleteMissing(ctx, tx, "audit_entries", auditIDs)
}

func deleteMissing(ctx context.Context, tx pgx.Tx, table string, ids []string) error {
	allowed := map[string]bool{"tax_periods": true, "transactions": true, "tax_declarations": true, "audit_entries": true}
	if !allowed[table] {
		return fmt.Errorf("invalid table %q", table)
	}
	if len(ids) == 0 {
		_, err := tx.Exec(ctx, "DELETE FROM "+table)
		return err
	}
	_, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE NOT (id = ANY($1::text[]))", ids)
	return err
}
