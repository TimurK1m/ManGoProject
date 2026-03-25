package repository

import (
	"Project/internal/model"
	"database/sql"
)

type ServiceRepository struct {
	db *sql.DB
}

func NewServiceRepository(db *sql.DB) *ServiceRepository {
	return &ServiceRepository{db: db}
}

func (r *ServiceRepository) Create(service model.Service) error {
	query := `
		INSERT INTO services (name, url, interval, user_id)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(query,
		service.Name,
		service.URL,
		service.Interval,
		service.UserID,
	)

	return err
}