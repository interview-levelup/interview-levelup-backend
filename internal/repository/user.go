package repository

import (
	"github.com/fan/interview-levelup-backend/internal/models"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(u *models.User) error {
	const q = `
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES (:id, :email, :password_hash, :created_at)`
	_, err := r.db.NamedExec(q, u)
	return err
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var u models.User
	err := r.db.Get(&u, `SELECT * FROM users WHERE email = $1`, email)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByID(id string) (*models.User, error) {
	var u models.User
	err := r.db.Get(&u, `SELECT * FROM users WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) UpdatePasswordHash(id, hash string) error {
	_, err := r.db.Exec(`UPDATE users SET password_hash = $1 WHERE id = $2`, hash, id)
	return err
}
