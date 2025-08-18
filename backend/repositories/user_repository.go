package repositories

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"tickets-by-uma/models"
)

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (email, name, created_at, updated_at) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id, created_at, updated_at`

	now := time.Now()
	return r.db.QueryRowx(query, user.Email, user.Name, now, now).StructScan(user)
}

func (r *userRepository) GetByID(id int) (*models.User, error) {
	user := &models.User{}
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.Get(user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.Get(user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *userRepository) Update(user *models.User) error {
	query := `
		UPDATE users 
		SET email = $1, name = $2, updated_at = $3 
		WHERE id = $4`

	user.UpdatedAt = time.Now()
	_, err := r.db.Exec(query, user.Email, user.Name, user.UpdatedAt, user.ID)
	return err
}

func (r *userRepository) Delete(id int) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}
