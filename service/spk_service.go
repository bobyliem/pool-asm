package service

import (
	"asm/models"
	"asm/repository"
)

type SpkService interface {
	GetSpkPerCust(tglAwal, tglAkhir, area, cust, purpose string) ([]models.SpkPerCustItem, error)
	GetFilterOptions(tglAwal, tglAkhir, area string) (*models.SpkFilterOptions, error)
}

type spkService struct {
	repo repository.SpkRepository
}

func NewSpkService(repo repository.SpkRepository) SpkService {
	return &spkService{repo: repo}
}

func (s *spkService) GetSpkPerCust(tglAwal, tglAkhir, area, cust, purpose string) ([]models.SpkPerCustItem, error) {
	return s.repo.GetSpkPerCust(tglAwal, tglAkhir, area, cust, purpose)
}

func (s *spkService) GetFilterOptions(tglAwal, tglAkhir, area string) (*models.SpkFilterOptions, error) {
	return s.repo.GetFilterOptions(tglAwal, tglAkhir, area)
}

