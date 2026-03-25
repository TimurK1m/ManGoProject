package service

import (
	"Project/internal/model"
	"Project/internal/repository"
)

type ServiceService struct {
	repo *repository.ServiceRepository
}

func NewServiceService(repo *repository.ServiceRepository) *ServiceService {
	return &ServiceService{repo: repo}
}

type CreateServiceInput struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Interval int    `json:"interval"`
}

func (s *ServiceService) Create(userID int, input CreateServiceInput) error {
	service := model.Service{
		Name:     input.Name,
		URL:      input.URL,
		Interval: input.Interval,
		UserID:   userID,
	}

	return s.repo.Create(service)
}