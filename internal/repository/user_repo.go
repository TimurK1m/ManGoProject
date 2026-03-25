package repository

import (
	"Project/internal/model" // 🔥 ВАЖНО
	"database/sql"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user model.User) error {
	query := `INSERT INTO users (email, password) VALUES ($1, $2)`
	_, err := r.db.Exec(query, user.Email, user.Password)
	return err
}

func (r *UserRepository) GetByEmail(email string) (model.User, error) {
	var user model.User

	query := `SELECT id, email, password FROM users WHERE email=$1`
	err := r.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Password)

	return user, err
}