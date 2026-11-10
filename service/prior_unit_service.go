package service

import (
	"asm/models"
	"asm/repository"
	"errors"
	"strings"
)

type PriorUnitService interface {
	GetAll(kategori, statue, search string) ([]models.PriorUnit, models.PriorUnitSummary, error)
	GetByID(id int) (*models.PriorUnit, error)
	Create(req *models.PriorUnitRequest) (*models.PriorUnit, error)
	Update(id int, req *models.PriorUnitRequest) (*models.PriorUnit, error)
	Delete(id int) error
}

type priorUnitService struct {
	repo repository.PriorUnitRepository
}

func NewPriorUnitService(repo repository.PriorUnitRepository) PriorUnitService {
	return &priorUnitService{repo: repo}
}

func (s *priorUnitService) GetAll(kategori, statue, search string) ([]models.PriorUnit, models.PriorUnitSummary, error) {
	return s.repo.GetAll(kategori, statue, search)
}

func (s *priorUnitService) GetByID(id int) (*models.PriorUnit, error) {
	return s.repo.GetByID(id)
}

func (s *priorUnitService) Create(req *models.PriorUnitRequest) (*models.PriorUnit, error) {
	if strings.TrimSpace(req.Nopol) == "" {
		return nil, errors.New("nopol wajib diisi")
	}

	statueVal := req.Statue
	if statueVal == "" {
		statueVal = "0"
	}

	unit := &models.PriorUnit{
		Nopol:    strings.TrimSpace(req.Nopol),
		Cabin:    strings.TrimSpace(req.Cabin),
		Jenis:    strings.TrimSpace(req.Jenis),
		Kategori: strings.ToLower(strings.TrimSpace(req.Kategori)),
		Statue:   statueVal,
		Note:     req.Note,
	}

	if err := s.repo.Create(unit); err != nil {
		return nil, err
	}
	return unit, nil
}

func (s *priorUnitService) Update(id int, req *models.PriorUnitRequest) (*models.PriorUnit, error) {
	unit, err := s.repo.GetByID(id)
	if err != nil {
		return nil, errors.New("data prior unit tidak ditemukan")
	}

	if strings.TrimSpace(req.Nopol) == "" {
		return nil, errors.New("nopol wajib diisi")
	}

	statueVal := req.Statue
	if statueVal == "" {
		statueVal = "0"
	}

	unit.Nopol = strings.TrimSpace(req.Nopol)
	unit.Cabin = strings.TrimSpace(req.Cabin)
	unit.Jenis = strings.TrimSpace(req.Jenis)
	unit.Kategori = strings.ToLower(strings.TrimSpace(req.Kategori))
	unit.Statue = statueVal
	unit.Note = req.Note

	if err := s.repo.Update(unit); err != nil {
		return nil, err
	}
	return unit, nil
}

func (s *priorUnitService) Delete(id int) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("data prior unit tidak ditemukan")
	}
	return s.repo.Delete(id)
}
