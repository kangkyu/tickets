package repositories

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	"tickets-by-uma/models"
)

type nwcConnectionRepository struct {
	db *sqlx.DB
}

func NewNWCConnectionRepository(db *sqlx.DB) NWCConnectionRepository {
	return &nwcConnectionRepository{db: db}
}

func (r *nwcConnectionRepository) Upsert(userID int, connectionURI string, expiresAt *time.Time) error {
	query := `
		INSERT INTO nwc_connections (user_id, connection_uri, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET connection_uri = $2, expires_at = $3, updated_at = $5`

	now := time.Now()
	_, err := r.db.Exec(query, userID, connectionURI, expiresAt, now, now)
	return err
}

func (r *nwcConnectionRepository) GetByUserID(userID int) (*models.NWCConnection, error) {
	conn := &models.NWCConnection{}
	query := `SELECT * FROM nwc_connections WHERE user_id = $1`
	err := r.db.Get(conn, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return conn, nil
}
