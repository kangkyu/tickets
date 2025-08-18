package repositories

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"tickets-by-uma/models"
)

type ticketRepository struct {
	db *sqlx.DB
}

func NewTicketRepository(db *sqlx.DB) TicketRepository {
	return &ticketRepository{db: db}
}

func (r *ticketRepository) Create(ticket *models.Ticket) error {
	query := `
		INSERT INTO tickets (event_id, user_id, ticket_code, payment_status, invoice_id, uma_address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	return r.db.QueryRowx(query,
		ticket.EventID, ticket.UserID, ticket.TicketCode, ticket.PaymentStatus,
		ticket.InvoiceID, ticket.UMAAddress, now, now).StructScan(ticket)
}

func (r *ticketRepository) GetByID(id int) (*models.Ticket, error) {
	ticket := &models.Ticket{}
	query := `SELECT * FROM tickets WHERE id = $1`
	err := r.db.Get(ticket, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return ticket, nil
}

func (r *ticketRepository) GetByTicketCode(ticketCode string) (*models.Ticket, error) {
	ticket := &models.Ticket{}
	query := `SELECT * FROM tickets WHERE ticket_code = $1`
	err := r.db.Get(ticket, query, ticketCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return ticket, nil
}

func (r *ticketRepository) GetByEventID(eventID int) ([]models.Ticket, error) {
	tickets := []models.Ticket{}
	query := `SELECT * FROM tickets WHERE event_id = $1 ORDER BY created_at DESC`
	err := r.db.Select(&tickets, query, eventID)
	return tickets, err
}

func (r *ticketRepository) GetByUserID(userID int) ([]models.Ticket, error) {
	tickets := []models.Ticket{}
	query := `SELECT * FROM tickets WHERE user_id = $1 ORDER BY created_at DESC`
	err := r.db.Select(&tickets, query, userID)
	return tickets, err
}

func (r *ticketRepository) GetByInvoiceID(invoiceID string) (*models.Ticket, error) {
	ticket := &models.Ticket{}
	query := `SELECT * FROM tickets WHERE invoice_id = $1`
	err := r.db.Get(ticket, query, invoiceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return ticket, nil
}

func (r *ticketRepository) Update(ticket *models.Ticket) error {
	query := `
		UPDATE tickets 
		SET event_id = $1, user_id = $2, ticket_code = $3, payment_status = $4, 
		    invoice_id = $5, uma_address = $6, paid_at = $7, updated_at = $8
		WHERE id = $9`

	ticket.UpdatedAt = time.Now()
	_, err := r.db.Exec(query,
		ticket.EventID, ticket.UserID, ticket.TicketCode, ticket.PaymentStatus,
		ticket.InvoiceID, ticket.UMAAddress, ticket.PaidAt, ticket.UpdatedAt, ticket.ID)
	return err
}

func (r *ticketRepository) UpdatePaymentStatus(id int, status string) error {
	query := `
		UPDATE tickets 
		SET payment_status = $1, updated_at = $2, paid_at = $3
		WHERE id = $4`

	now := time.Now()
	var paidAt *time.Time
	if status == "paid" {
		paidAt = &now
	}

	_, err := r.db.Exec(query, status, now, paidAt, id)
	return err
}

func (r *ticketRepository) GetPendingTickets() ([]models.Ticket, error) {
	tickets := []models.Ticket{}
	query := `SELECT * FROM tickets WHERE payment_status = 'pending' ORDER BY created_at ASC`
	err := r.db.Select(&tickets, query)
	return tickets, err
}

func (r *ticketRepository) CountByEventAndStatus(eventID int, status string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM tickets WHERE event_id = $1 AND payment_status = $2`
	err := r.db.Get(&count, query, eventID, status)
	return count, err
}

// HasUserTicketForEvent checks if a user has any tickets for a specific event
func (r *ticketRepository) HasUserTicketForEvent(userID, eventID int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM tickets WHERE user_id = $1 AND event_id = $2`
	err := r.db.Get(&count, query, userID, eventID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
