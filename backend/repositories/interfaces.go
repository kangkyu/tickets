package repositories

import (
	"tickets-by-uma/models"
)

// UserRepository defines operations for user data
type UserRepository interface {
	Create(user *models.User) error
	GetByID(id int) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Update(user *models.User) error
	Delete(id int) error
}

// EventRepository defines operations for event data
type EventRepository interface {
	Create(event *models.Event) error
	GetByID(id int) (*models.Event, error)
	GetByIDWithUMAInvoice(id int) (*models.Event, error)
	GetAll(limit, offset int) ([]models.Event, error)
	GetActive(limit, offset int) ([]models.Event, error)
	Update(event *models.Event) error
	Delete(id int) error
	GetAvailableTicketCount(eventID int) (int, error)
	UpdateCapacity(eventID, newCapacity int) error
}

type UMARequestInvoiceRepository interface {
	Create(invoice *models.UMARequestInvoice) error
	GetByEventID(eventID int) (*models.UMARequestInvoice, error)
	GetByTicketID(ticketID int) (*models.UMARequestInvoice, error)
	Update(invoice *models.UMARequestInvoice) error
	Delete(id int) error
}

// TicketRepository defines operations for ticket data
type TicketRepository interface {
	Create(ticket *models.Ticket) error
	GetByID(id int) (*models.Ticket, error)
	GetByTicketCode(ticketCode string) (*models.Ticket, error)
	GetByEventID(eventID int) ([]models.Ticket, error)
	GetByUserID(userID int) ([]models.Ticket, error)
	GetByInvoiceID(invoiceID string) (*models.Ticket, error)
	Update(ticket *models.Ticket) error
	UpdatePaymentStatus(id int, status string) error
	GetPendingTickets() ([]models.Ticket, error)
	CountByEventAndStatus(eventID int, status string) (int, error)
	HasUserTicketForEvent(userID, eventID int) (bool, error)
}

// PaymentRepository defines operations for payment data
type PaymentRepository interface {
	Create(payment *models.Payment) error
	GetByID(id int) (*models.Payment, error)
	GetByInvoiceID(invoiceID string) (*models.Payment, error)
	GetByTicketID(ticketID int) (*models.Payment, error)
	Update(payment *models.Payment) error
	UpdateStatus(id int, status string) error
	GetPendingPayments() ([]models.Payment, error)
	GetAvailablePaymentForEvent(eventID int) (*models.Payment, error)
}
