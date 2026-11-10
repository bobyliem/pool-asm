package service

import (
	"asm/models"
	"asm/repository"
)

type CsvSpkService interface {
	GetFilterOptions() (*models.CsvSpkFilterOptions, error)
	GetSpkPerCust(area, customer, purpose, tglAwal, tglAkhir string) ([]models.CsvSpkPerCustItem, error)
	GetSpkDetail(nopol, tglAwal, tglAkhir string) (*models.CsvSpkDetailResponse, error)
	GetSummaryPerCust(area, customer, purpose, tglAwal, tglAkhir string) (*models.CsvSummaryPerCustResponse, error)
	GetResumePerKategori(area, tglAwal, tglAkhir string) (*models.CsvResumePerKategoriResponse, error)
	GetResumeKategoriDetail(area, customer, category, status, tglAwal, tglAkhir string) ([]models.CsvResumeKategoriDetailItem, error)
}

type csvSpkService struct {
	repo repository.CsvSpkRepository
}

func NewCsvSpkService(repo repository.CsvSpkRepository) CsvSpkService {
	return &csvSpkService{repo: repo}
}

func (s *csvSpkService) GetFilterOptions() (*models.CsvSpkFilterOptions, error) {
	return s.repo.GetFilterOptions()
}

func (s *csvSpkService) GetSpkPerCust(area, customer, purpose, tglAwal, tglAkhir string) ([]models.CsvSpkPerCustItem, error) {
	return s.repo.GetSpkPerCust(area, customer, purpose, tglAwal, tglAkhir)
}

func (s *csvSpkService) GetSpkDetail(nopol, tglAwal, tglAkhir string) (*models.CsvSpkDetailResponse, error) {
	return s.repo.GetSpkDetail(nopol, tglAwal, tglAkhir)
}

func (s *csvSpkService) GetSummaryPerCust(area, customer, purpose, tglAwal, tglAkhir string) (*models.CsvSummaryPerCustResponse, error) {
	return s.repo.GetSummaryPerCust(area, customer, purpose, tglAwal, tglAkhir)
}

func (s *csvSpkService) GetResumePerKategori(area, tglAwal, tglAkhir string) (*models.CsvResumePerKategoriResponse, error) {
	return s.repo.GetResumePerKategori(area, tglAwal, tglAkhir)
}

func (s *csvSpkService) GetResumeKategoriDetail(area, customer, category, status, tglAwal, tglAkhir string) ([]models.CsvResumeKategoriDetailItem, error) {
	return s.repo.GetResumeKategoriDetail(area, customer, category, status, tglAwal, tglAkhir)
}
