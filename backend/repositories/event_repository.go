package repositories

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"tickets-by-uma/models"
)

type eventRepository struct {
	db      *sqlx.DB
	umaRepo UMARequestInvoiceRepository
}

func NewEventRepository(db *sqlx.DB) EventRepository {
	return &eventRepository{
		db:      db,
		umaRepo: NewUMARequestInvoiceRepository(db),
	}
}

func (r *eventRepository) Create(event *models.Event) error {
	query := `
		INSERT INTO events (title, description, start_time, end_time, capacity, price_sats, stream_url, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	return r.db.QueryRowx(query,
		event.Title, event.Description, event.StartTime,
		event.EndTime, event.Capacity, event.PriceSats, event.StreamURL, event.IsActive, now, now).StructScan(event)
}

func (r *eventRepository) GetByID(id int) (*models.Event, error) {
	event := &models.Event{}
	query := `SELECT * FROM events WHERE id = $1`
	err := r.db.Get(event, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return event, nil
}

func (r *eventRepository) GetByIDWithUMAInvoice(id int) (*models.Event, error) {
	event := &models.Event{}
	query := `SELECT * FROM events WHERE id = $1`

	err := r.db.Get(event, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Fetch UMA Request invoice for this event and populate the relationship
	umaInvoice, err := r.umaRepo.GetByEventID(id)
	if err != nil {
		// Log error but don't fail the request
		// The event will be returned without UMA invoice data
	} else {
		event.UMARequestInvoice = umaInvoice
	}

	return event, nil
}

func (r *eventRepository) GetAll(limit, offset int) ([]models.Event, error) {
	events := []models.Event{}
	query := `
		SELECT e.id, e.title, e.description, e.start_time, e.end_time, 
		       e.capacity, e.price_sats, e.stream_url, e.is_active, 
		       e.created_at, e.updated_at
		FROM events e
		ORDER BY e.start_time ASC 
		LIMIT $1 OFFSET $2`

	err := r.db.Select(&events, query, limit, offset)
	if err != nil {
		return nil, err
	}

	// Now fetch UMA Request invoices for each event and populate the relationship
	for i := range events {
		umaInvoice, err := r.umaRepo.GetByEventID(events[i].ID)
		if err != nil {
			// Log error but don't fail the entire request
			continue
		}
		events[i].UMARequestInvoice = umaInvoice
	}

	return events, nil
}

func (r *eventRepository) GetActive(limit, offset int) ([]models.Event, error) {
	events := []models.Event{}
	query := `
		SELECT e.id, e.title, e.description, e.start_time, e.end_time, 
		       e.capacity, e.price_sats, e.stream_url, e.is_active, 
		       e.created_at, e.updated_at
		FROM events e
		WHERE e.is_active = true 
		ORDER BY e.start_time ASC 
		LIMIT $1 OFFSET $2`

	err := r.db.Select(&events, query, limit, offset)
	if err != nil {
		return nil, err
	}

	// Now fetch UMA Request invoices for each event and populate the relationship
	for i := range events {
		umaInvoice, err := r.umaRepo.GetByEventID(events[i].ID)
		if err != nil {
			// Log error but don't fail the entire request
			continue
		}
		events[i].UMARequestInvoice = umaInvoice
	}

	return events, nil
}

func (r *eventRepository) Update(event *models.Event) error {
	query := `
		UPDATE events 
		SET title = $1, description = $2, start_time = $3, end_time = $4, 
		    capacity = $5, price_sats = $6, stream_url = $7, is_active = $8, updated_at = $9
		WHERE id = $10`

	event.UpdatedAt = time.Now()
	_, err := r.db.Exec(query,
		event.Title, event.Description, event.StartTime, event.EndTime,
		event.Capacity, event.PriceSats, event.StreamURL, event.IsActive, event.UpdatedAt, event.ID)
	return err
}

func (r *eventRepository) Delete(id int) error {
	query := `DELETE FROM events WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *eventRepository) GetAvailableTicketCount(eventID int) (int, error) {
	var count int
	query := `
		SELECT (e.capacity - COALESCE(COUNT(t.id), 0)) as available
		FROM events e
		LEFT JOIN tickets t ON e.id = t.event_id AND t.payment_status = 'paid'
		WHERE e.id = $1
		GROUP BY e.capacity`

	err := r.db.Get(&count, query, eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			// If no tickets sold, return full capacity
			var capacity int
			err = r.db.Get(&capacity, "SELECT capacity FROM events WHERE id = $1", eventID)
			if err != nil {
				return 0, err
			}
			return capacity, nil
		}
		return 0, err
	}
	return count, nil
}

func (r *eventRepository) UpdateCapacity(eventID, newCapacity int) error {
	query := `UPDATE events SET capacity = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(query, newCapacity, time.Now(), eventID)
	return err
}

// UMARequestInvoiceRepository implementation
type umaRequestInvoiceRepository struct {
	db *sqlx.DB
}

func NewUMARequestInvoiceRepository(db *sqlx.DB) UMARequestInvoiceRepository {
	return &umaRequestInvoiceRepository{db: db}
}

func (r *umaRequestInvoiceRepository) Create(invoice *models.UMARequestInvoice) error {
	query := `
		INSERT INTO uma_request_invoices (event_id, invoice_id, payment_hash, bolt11, amount_sats, status, uma_address, description, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	return r.db.QueryRowx(query,
		invoice.EventID, invoice.InvoiceID, invoice.PaymentHash, invoice.Bolt11,
		invoice.AmountSats, invoice.Status, invoice.UMAAddress, invoice.Description,
		invoice.ExpiresAt, now, now).StructScan(invoice)
}

func (r *umaRequestInvoiceRepository) GetByEventID(eventID int) (*models.UMARequestInvoice, error) {
	invoice := &models.UMARequestInvoice{}
	query := `SELECT * FROM uma_request_invoices WHERE event_id = $1`
	err := r.db.Get(invoice, query, eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No invoice found for this event
		}
		return nil, err
	}
	return invoice, nil
}

func (r *umaRequestInvoiceRepository) Update(invoice *models.UMARequestInvoice) error {
	query := `
		UPDATE uma_request_invoices 
		SET invoice_id = $1, payment_hash = $2, bolt11 = $3, amount_sats = $4, 
		    status = $5, uma_address = $6, description = $7, expires_at = $8, updated_at = $9
		WHERE id = $10`

	invoice.UpdatedAt = time.Now()
	_, err := r.db.Exec(query,
		invoice.InvoiceID, invoice.PaymentHash, invoice.Bolt11, invoice.AmountSats,
		invoice.Status, invoice.UMAAddress, invoice.Description, invoice.ExpiresAt,
		invoice.UpdatedAt, invoice.ID)
	return err
}

func (r *umaRequestInvoiceRepository) Delete(id int) error {
	query := `DELETE FROM uma_request_invoices WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}
