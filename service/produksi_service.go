package service

import (
	"asm/models"
	"asm/repository"
)

type ProduksiService interface {
	GetResumePerHari(tglAwal, tglAkhir string) (*models.ResumePerHariResponse, error)
	GetSummaryPerDriver(tglAwal, tglAkhir string) (*models.SummaryPerDriverResponse, error)
	GetMonitoringRitase(tglAwal, tglAkhir string) (*models.MonitoringRitaseResponse, error)
	GetTransferOut(filter models.TransferOutFilter) (*models.TransferOutResponse, error)
	GetTransferOutFilterOptions() (*models.TransferOutFilterOptions, error)
}

type produksiService struct {
	repo repository.ProduksiRepository
}

func NewProduksiService(repo repository.ProduksiRepository) ProduksiService {
	return &produksiService{repo: repo}
}

func (s *produksiService) GetResumePerHari(tglAwal, tglAkhir string) (*models.ResumePerHariResponse, error) {
	return s.repo.GetResumePerHari(tglAwal, tglAkhir)
}

func (s *produksiService) GetSummaryPerDriver(tglAwal, tglAkhir string) (*models.SummaryPerDriverResponse, error) {
	return s.repo.GetSummaryPerDriver(tglAwal, tglAkhir)
}

func (s *produksiService) GetMonitoringRitase(tglAwal, tglAkhir string) (*models.MonitoringRitaseResponse, error) {
	return s.repo.GetMonitoringRitase(tglAwal, tglAkhir)
}

func (s *produksiService) GetTransferOut(filter models.TransferOutFilter) (*models.TransferOutResponse, error) {
	return s.repo.GetTransferOut(filter)
}

func (s *produksiService) GetTransferOutFilterOptions() (*models.TransferOutFilterOptions, error) {
	return s.repo.GetTransferOutFilterOptions()
}
