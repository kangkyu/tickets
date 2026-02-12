package repositories

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	"tickets-by-uma/models"
)

type paymentRepository struct {
	db *sqlx.DB
}

func NewPaymentRepository(db *sqlx.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(payment *models.Payment) error {
	query := `
		INSERT INTO payments (ticket_id, invoice_id, amount_sats, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	return r.db.QueryRowx(query,
		payment.TicketID, payment.InvoiceID, payment.Amount, payment.Status, now, now).StructScan(payment)
}

func (r *paymentRepository) GetByID(id int) (*models.Payment, error) {
	payment := &models.Payment{}
	query := `SELECT * FROM payments WHERE id = $1`
	err := r.db.Get(payment, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return payment, nil
}

func (r *paymentRepository) GetByInvoiceID(invoiceID string) (*models.Payment, error) {
	payment := &models.Payment{}
	query := `SELECT * FROM payments WHERE invoice_id = $1`
	err := r.db.Get(payment, query, invoiceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return payment, nil
}

func (r *paymentRepository) GetByTicketID(ticketID int) (*models.Payment, error) {
	payment := &models.Payment{}
	query := `SELECT * FROM payments WHERE ticket_id = $1`
	err := r.db.Get(payment, query, ticketID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return payment, nil
}

func (r *paymentRepository) Update(payment *models.Payment) error {
	query := `
		UPDATE payments 
		SET ticket_id = $1, invoice_id = $2, amount_sats = $3, status = $4, paid_at = $5, updated_at = $6
		WHERE id = $7`

	payment.UpdatedAt = time.Now()
	_, err := r.db.Exec(query,
		payment.TicketID, payment.InvoiceID, payment.Amount, payment.Status,
		payment.PaidAt, payment.UpdatedAt, payment.ID)
	return err
}

func (r *paymentRepository) UpdateStatus(id int, status string) error {
	query := `
		UPDATE payments 
		SET status = $1, updated_at = $2, paid_at = $3
		WHERE id = $4`

	now := time.Now()
	var paidAt *time.Time
	if status == "paid" {
		paidAt = &now
	}

	_, err := r.db.Exec(query, status, now, paidAt, id)
	return err
}

func (r *paymentRepository) GetPendingPayments() ([]models.Payment, error) {
	payments := []models.Payment{}
	query := `SELECT * FROM payments WHERE status = 'pending' ORDER BY created_at ASC`
	err := r.db.Select(&payments, query)
	return payments, err
}

func (r *paymentRepository) GetAvailablePaymentForEvent(eventID int) (*models.Payment, error) {
	// This method is no longer needed with UMA Request pattern
	// Return nil to indicate no pre-created payments available
	return nil, nil
}

func (r *paymentRepository) GetOldestPendingByAmount(amountSats int64) (*models.Payment, error) {
	payment := &models.Payment{}
	query := `SELECT * FROM payments WHERE status = 'pending' AND amount_sats = $1 ORDER BY created_at ASC LIMIT 1`
	err := r.db.Get(payment, query, amountSats)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return payment, nil
}
