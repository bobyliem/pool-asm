package service

import (
	"asm/models"
	"asm/repository"
)

type SolarService interface {
	GetPemakaianSolar(filter models.PemakaianSolarFilter) (map[string]interface{}, error)
	GetFilterOptions() (map[string]interface{}, error)
}

type solarService struct {
	repo repository.SolarRepository
}

func NewSolarService(repo repository.SolarRepository) SolarService {
	return &solarService{repo: repo}
}

func (s *solarService) GetPemakaianSolar(filter models.PemakaianSolarFilter) (map[string]interface{}, error) {
	data, totalRows, err := s.repo.GetPemakaianSolar(filter)
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.GetSummary(filter)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"data":       data,
		"total_rows": totalRows,
		"page":       filter.Page,
		"limit":      filter.Limit,
		"summary":    summary,
	}, nil
}

func (s *solarService) GetFilterOptions() (map[string]interface{}, error) {
	units, shifts, minTgl, maxTgl, err := s.repo.GetFilterOptions()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"units":   units,
		"shifts":  shifts,
		"min_tgl": minTgl,
		"max_tgl": maxTgl,
	}, nil
}
