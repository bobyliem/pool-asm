package repository

import (
	"math"
	"sort"
	"strings"
	"time"

	"asm/models"
	"gorm.io/gorm"
)

type ProduksiRepository interface {
	GetResumePerHari(tglAwal, tglAkhir string) (*models.ResumePerHariResponse, error)
	GetSummaryPerDriver(tglAwal, tglAkhir string) (*models.SummaryPerDriverResponse, error)
	GetMonitoringRitase(tglAwal, tglAkhir string) (*models.MonitoringRitaseResponse, error)
	GetTransferOut(filter models.TransferOutFilter) (*models.TransferOutResponse, error)
	GetTransferOutFilterOptions() (*models.TransferOutFilterOptions, error)
}

type produksiRepository struct {
	db *gorm.DB
}

func NewProduksiRepository(db *gorm.DB) ProduksiRepository {
	return &produksiRepository{db: db}
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}

func (r *produksiRepository) GetResumePerHari(tglAwal, tglAkhir string) (*models.ResumePerHariResponse, error) {
	var items []models.ResumePerHariItem

	query := `
		SELECT DATE_FORMAT(tgl, '%Y-%m-%d') AS tgl
			, SUM(jml_rit)AS jmlRit, COUNT(nopol)AS jmlUnit
			, ROUND((SUM(jml_rit)/COUNT(nopol)),2)ritPerUnit
			, ROUND(SUM(qty_awal),2)AS ttlMuat, ROUND(SUM(qty_akhir),2)AS ttlSetor
			, ROUND(SUM(waste),2)AS totalWaste
			, ROUND(SUM(qty_akhir)/SUM(jml_rit),2) AS rataPerRit
			, ROUND(SUM(waste)/SUM(jml_rit),2) AS wastePerRit
		FROM(
			SELECT tgl, nopol, sopir, SUM(cnt)AS jml_rit, ROUND(SUM(qty_src),2)AS qty_awal, ROUND(SUM(qty_dest),2)AS qty_akhir
				, ROUND((SUM(qty_src) - SUM(qty_dest)),2) AS waste
			FROM(
				SELECT DATE(tgl_sj)AS tgl, nopol, sopir
					, qty_src, qty_dest, 1 AS cnt
				FROM ss_pok
				WHERE qty_src <> 0
					AND DATE(tgl_sj) BETWEEN ? AND ?
			)AS dt
			GROUP BY tgl, nopol, sopir
		)AS dt
		GROUP BY tgl
		ORDER BY tgl DESC
	`

	if err := r.db.Raw(query, tglAwal, tglAkhir).Scan(&items).Error; err != nil {
		return nil, err
	}

	// Hitung Total Unit Unik (Distinct nopol) selama periode yang dipilih
	var totalUnitUnique int
	_ = r.db.Raw(`SELECT COUNT(DISTINCT nopol) FROM ss_pok WHERE qty_src <> 0 AND DATE(tgl_sj) BETWEEN ? AND ?`, tglAwal, tglAkhir).Scan(&totalUnitUnique).Error

	var totalRit int
	var totalMuat, totalSetor, totalWaste float64

	dailyTarget := 365.0
	for i := range items {
		items[i].TargetTon = dailyTarget
		items[i].SelisihTon = round2(items[i].TtlSetor - dailyTarget)
		if dailyTarget > 0 {
			items[i].Pencapaian = round2((items[i].TtlSetor / dailyTarget) * 100)
		}

		totalRit += items[i].JmlRit
		totalMuat += items[i].TtlMuat
		totalSetor += items[i].TtlSetor
		totalWaste += items[i].TotalWaste
	}

	totalHari := len(items)
	var avgRitPerHari, totalTarget, totalSelisih, avgSetorPerHari, kekuranganPerHari, pencapaianPersen float64
	if totalHari > 0 {
		avgRitPerHari = round2(float64(totalRit) / float64(totalHari))
		totalTarget = round2(dailyTarget * float64(totalHari))
		totalSelisih = round2(totalSetor - totalTarget)
		avgSetorPerHari = round2(totalSetor / float64(totalHari))
		kekuranganPerHari = round2(dailyTarget - avgSetorPerHari)
		if totalTarget > 0 {
			pencapaianPersen = round2((totalSetor / totalTarget) * 100)
		}
	}

	var avgRitPerUnit float64
	if totalUnitUnique > 0 {
		avgRitPerUnit = round2(float64(totalRit) / float64(totalUnitUnique))
	}

	var avgSetorPerRit, avgWastePerRit float64
	if totalRit > 0 {
		avgSetorPerRit = round2(totalSetor / float64(totalRit))
		avgWastePerRit = round2(totalWaste / float64(totalRit))
	}

	var wastePercentage float64
	if totalMuat > 0 {
		wastePercentage = round2((totalWaste / totalMuat) * 100)
	}

	var lastUpdate string
	_ = r.db.Raw("SELECT DATE_FORMAT(MAX(finished), '%d-%b-%Y %H:%i:%s') FROM log_imp").Scan(&lastUpdate).Error

	summary := models.ResumePerHariSummary{
		TotalHari:         totalHari,
		TotalRit:          totalRit,
		AvgRitPerHari:     avgRitPerHari,
		TotalUnitUnique:   totalUnitUnique,
		AvgRitPerUnit:     avgRitPerUnit,
		TotalMuat:         round2(totalMuat),
		TotalSetor:        round2(totalSetor),
		TotalWaste:        round2(totalWaste),
		AvgSetorPerRit:    avgSetorPerRit,
		AvgWastePerRit:    avgWastePerRit,
		WastePercentage:   wastePercentage,
		DailyTarget:       dailyTarget,
		TotalTarget:       totalTarget,
		TotalSelisih:      totalSelisih,
		AvgSetorPerHari:   avgSetorPerHari,
		KekuranganPerHari: kekuranganPerHari,
		PencapaianPersen:  pencapaianPersen,
	}

	return &models.ResumePerHariResponse{
		TglAwal:    tglAwal,
		TglAkhir:   tglAkhir,
		LastUpdate: lastUpdate,
		Summary:    summary,
		Items:      items,
	}, nil
}

func (r *produksiRepository) GetSummaryPerDriver(tglAwal, tglAkhir string) (*models.SummaryPerDriverResponse, error) {
	var items []models.SummaryPerDriverItem

	query := `
		SELECT sopir
			, SUM(jml_rit)AS jmlRit, COUNT(nopol)AS jmlHari
			, ROUND((SUM(jml_rit)/COUNT(nopol)),2)ritPerHari
			, ROUND(SUM(qty_awal),2)AS ttlMuat, ROUND(SUM(qty_akhir),2)AS ttlSetor
			, ROUND(SUM(waste),2)AS totalWaste
			, ROUND(SUM(qty_akhir)/SUM(jml_rit),2) AS rataPerRit
			, ROUND(SUM(waste)/SUM(jml_rit),2) AS wastePerRit
		FROM(
			SELECT tgl, nopol, sopir, SUM(cnt)AS jml_rit, ROUND(SUM(qty_src),2)AS qty_awal, ROUND(SUM(qty_dest),2)AS qty_akhir
				, ROUND((SUM(qty_src) - SUM(qty_dest)),2) AS waste
			FROM(
				SELECT DATE(tgl_sj)AS tgl, nopol, sopir
					, qty_src, qty_dest, 1 AS cnt
				FROM ss_pok
				WHERE qty_src <> 0
					AND DATE(tgl_sj) BETWEEN ? AND ?
			)AS dt
			GROUP BY tgl, nopol, sopir
		)AS dt
		GROUP BY sopir
		ORDER BY jmlRit DESC
	`

	if err := r.db.Raw(query, tglAwal, tglAkhir).Scan(&items).Error; err != nil {
		return nil, err
	}

	totalSopir := len(items)
	var totalRit int
	var totalMuat, totalSetor, totalWaste float64

	for _, it := range items {
		totalRit += it.JmlRit
		totalMuat += it.TtlMuat
		totalSetor += it.TtlSetor
		totalWaste += it.TotalWaste
	}

	var avgRitPerDriver float64
	if totalSopir > 0 {
		avgRitPerDriver = round2(float64(totalRit) / float64(totalSopir))
	}

	var avgSetorPerRit, avgWastePerRit float64
	if totalRit > 0 {
		avgSetorPerRit = round2(totalSetor / float64(totalRit))
		avgWastePerRit = round2(totalWaste / float64(totalRit))
	}

	var wastePercentage float64
	if totalMuat > 0 {
		wastePercentage = round2((totalWaste / totalMuat) * 100)
	}

	var lastUpdate string
	_ = r.db.Raw("SELECT DATE_FORMAT(MAX(finished), '%d-%b-%Y %H:%i:%s') FROM log_imp").Scan(&lastUpdate).Error

	summary := models.SummaryPerDriverSummary{
		TotalSopir:      totalSopir,
		TotalRit:        totalRit,
		AvgRitPerDriver: avgRitPerDriver,
		TotalMuat:       round2(totalMuat),
		TotalSetor:      round2(totalSetor),
		TotalWaste:      round2(totalWaste),
		AvgSetorPerRit:  avgSetorPerRit,
		AvgWastePerRit:  avgWastePerRit,
		WastePercentage: wastePercentage,
	}

	return &models.SummaryPerDriverResponse{
		TglAwal:    tglAwal,
		TglAkhir:   tglAkhir,
		LastUpdate: lastUpdate,
		Summary:    summary,
		Items:      items,
	}, nil
}

type rawMonitoringItem struct {
	Tgl      string  `gorm:"column:tgl"`
	Nopol    string  `gorm:"column:nopol"`
	Sopir    string  `gorm:"column:sopir"`
	AmbilDo  int     `gorm:"column:ambil_do"`
	VoidRit  int     `gorm:"column:void_rit"`
	SjRit    int     `gorm:"column:sj_rit"`
	VolAsal  float64 `gorm:"column:vol_asal"`
	VolAkhir float64 `gorm:"column:vol_akhir"`
	Selisih  float64 `gorm:"column:selisih"`
}

func (r *produksiRepository) GetMonitoringRitase(tglAwal, tglAkhir string) (*models.MonitoringRitaseResponse, error) {
	// Parse rentang tanggal
	t1, err := time.Parse("2006-01-02", tglAwal)
	if err != nil {
		t1 = time.Now()
		tglAwal = t1.Format("2006-01-02")
	}
	t2, err := time.Parse("2006-01-02", tglAkhir)
	if err != nil {
		t2 = time.Now()
		tglAkhir = t2.Format("2006-01-02")
	}
	if t1.After(t2) {
		t1, t2 = t2, t1
		tglAwal, tglAkhir = tglAkhir, tglAwal
	}

	// Generate list dates berurutan (YYYY-MM-DD)
	var dates []string
	for cur := t1; !cur.After(t2); cur = cur.AddDate(0, 0, 1) {
		dates = append(dates, cur.Format("2006-01-02"))
	}

	// Ambil semua master armada Nopol dari ss_pok
	var allNopol []string
	if err := r.db.Raw(`SELECT DISTINCT nopol FROM ss_pok WHERE nopol IS NOT NULL AND nopol <> '' ORDER BY nopol`).Scan(&allNopol).Error; err != nil {
		return nil, err
	}

	// Query agregat per (tgl, nopol)
	query := `
		SELECT 
			DATE_FORMAT(tgl_sj, '%Y-%m-%d') AS tgl,
			nopol,
			MAX(sopir) AS sopir,
			COUNT(*) AS ambil_do,
			SUM(CASE WHEN statue LIKE '%Void%' THEN 1 ELSE 0 END) AS void_rit,
			SUM(CASE WHEN qty_src <> 0 THEN 1 ELSE 0 END) AS sj_rit,
			ROUND(SUM(CASE WHEN qty_src <> 0 THEN qty_src ELSE 0 END), 2) AS vol_asal,
			ROUND(SUM(CASE WHEN qty_src <> 0 THEN qty_dest ELSE 0 END), 2) AS vol_akhir,
			ROUND(SUM(CASE WHEN qty_src <> 0 THEN (qty_dest - qty_src) ELSE 0 END), 2) AS selisih
		FROM ss_pok
		WHERE DATE(tgl_sj) BETWEEN ? AND ?
		GROUP BY DATE_FORMAT(tgl_sj, '%Y-%m-%d'), nopol
		ORDER BY nopol, tgl
	`
	var rawData []rawMonitoringItem
	if err := r.db.Raw(query, tglAwal, tglAkhir).Scan(&rawData).Error; err != nil {
		return nil, err
	}

	// Peta data: nopol -> map[tgl]MonitoringRitaseCell
	nopolDataMap := make(map[string]map[string]models.MonitoringRitaseCell)
	nopolSopirMap := make(map[string]string)
	nopolRitSum := make(map[string]int)

	for _, item := range rawData {
		if _, exists := nopolDataMap[item.Nopol]; !exists {
			nopolDataMap[item.Nopol] = make(map[string]models.MonitoringRitaseCell)
		}
		nopolDataMap[item.Nopol][item.Tgl] = models.MonitoringRitaseCell{
			AmbilDo:  item.AmbilDo,
			VoidRit:  item.VoidRit,
			SjRit:    item.SjRit,
			VolAsal:  item.VolAsal,
			VolAkhir: item.VolAkhir,
			Selisih:  item.Selisih,
		}
		if item.Sopir != "" {
			nopolSopirMap[item.Nopol] = item.Sopir
		}
		nopolRitSum[item.Nopol] += item.AmbilDo
	}

	// Buat baris untuk setiap nopol
	var rows []models.MonitoringRitaseRow
	for _, np := range allNopol {
		dMap, exists := nopolDataMap[np]
		if !exists {
			dMap = make(map[string]models.MonitoringRitaseCell)
		}
		// Pastikan semua tanggal terisi minimal object kosong jika tidak ada data
		for _, d := range dates {
			if _, hasDate := dMap[d]; !hasDate {
				dMap[d] = models.MonitoringRitaseCell{
					AmbilDo:  0,
					VoidRit:  0,
					SjRit:    0,
					VolAsal:  0,
					VolAkhir: 0,
					Selisih:  0,
				}
			}
		}

		rows = append(rows, models.MonitoringRitaseRow{
			Nopol: np,
			Sopir: nopolSopirMap[np],
			Data:  dMap,
		})
	}

	// Urutkan rows: unit yang aktif (ada rit) di atas, unit yang kosong di bawah
	sort.SliceStable(rows, func(i, j int) bool {
		sumI := nopolRitSum[rows[i].Nopol]
		sumJ := nopolRitSum[rows[j].Nopol]
		if sumI != sumJ {
			return sumI > sumJ // Terbanyak ke terkecil
		}
		return rows[i].Nopol < rows[j].Nopol
	})

	// Hitung Totals per tanggal dan GrandTotal
	totals := make(map[string]models.MonitoringRitaseTotal)
	var grandTotal models.MonitoringRitaseTotal

	for _, d := range dates {
		var tot models.MonitoringRitaseTotal
		for _, rItem := range rows {
			cell := rItem.Data[d]
			tot.AmbilDo += cell.AmbilDo
			tot.VoidRit += cell.VoidRit
			tot.SjRit += cell.SjRit
			tot.VolAsal += cell.VolAsal
			tot.VolAkhir += cell.VolAkhir
			tot.Selisih += cell.Selisih
		}
		tot.VolAsal = round2(tot.VolAsal)
		tot.VolAkhir = round2(tot.VolAkhir)
		tot.Selisih = round2(tot.Selisih)
		totals[d] = tot

		grandTotal.AmbilDo += tot.AmbilDo
		grandTotal.VoidRit += tot.VoidRit
		grandTotal.SjRit += tot.SjRit
		grandTotal.VolAsal += tot.VolAsal
		grandTotal.VolAkhir += tot.VolAkhir
		grandTotal.Selisih += tot.Selisih
	}
	grandTotal.VolAsal = round2(grandTotal.VolAsal)
	grandTotal.VolAkhir = round2(grandTotal.VolAkhir)
	grandTotal.Selisih = round2(grandTotal.Selisih)

	var lastUpdate string
	_ = r.db.Raw("SELECT DATE_FORMAT(MAX(finished), '%d-%b-%Y %H:%i:%s') FROM log_imp").Scan(&lastUpdate).Error

	return &models.MonitoringRitaseResponse{
		TglAwal:    tglAwal,
		TglAkhir:   tglAkhir,
		LastUpdate: lastUpdate,
		Dates:      dates,
		Rows:       rows,
		Totals:     totals,
		GrandTotal: grandTotal,
	}, nil
}

func (r *produksiRepository) GetTransferOutFilterOptions() (*models.TransferOutFilterOptions, error) {
	var products, nopols, destinations []string

	_ = r.db.Raw(`
		SELECT DISTINCT TRIM(prd) AS prd
		FROM api_stockpile
		WHERE dk IN ('Out', 'Transfer Out')
		  AND TRIM(prd) <> '' AND TRIM(prd) <> '0'
		ORDER BY prd ASC
	`).Pluck("prd", &products).Error

	_ = r.db.Raw(`
		SELECT DISTINCT TRIM(nopol) AS nopol
		FROM api_stockpile
		WHERE dk IN ('Out', 'Transfer Out')
		  AND TRIM(nopol) <> '' AND TRIM(nopol) <> '0'
		ORDER BY nopol ASC
	`).Pluck("nopol", &nopols).Error

	_ = r.db.Raw(`
		SELECT DISTINCT TRIM(dest) AS dest
		FROM api_stockpile
		WHERE dk IN ('Out', 'Transfer Out')
		  AND TRIM(dest) <> '' AND TRIM(dest) <> '0'
		ORDER BY dest ASC
	`).Pluck("dest", &destinations).Error

	if products == nil {
		products = []string{}
	}
	if nopols == nil {
		nopols = []string{}
	}
	if destinations == nil {
		destinations = []string{}
	}

	return &models.TransferOutFilterOptions{
		Products:     products,
		Nopols:       nopols,
		Destinations: destinations,
	}, nil
}

func (r *produksiRepository) GetTransferOut(filter models.TransferOutFilter) (*models.TransferOutResponse, error) {
	whereClauses := []string{
		"dk IN ('Out', 'Transfer Out')",
		"DATE(tgl) BETWEEN ? AND ?",
	}
	args := []interface{}{filter.TglAwal, filter.TglAkhir}

	prd := strings.TrimSpace(filter.Prd)
	if prd != "" && !strings.EqualFold(prd, "all") {
		whereClauses = append(whereClauses, "TRIM(prd) = ?")
		args = append(args, prd)
	}

	nopol := strings.TrimSpace(filter.Nopol)
	if nopol != "" && !strings.EqualFold(nopol, "all") {
		whereClauses = append(whereClauses, "TRIM(nopol) LIKE ?")
		args = append(args, "%"+nopol+"%")
	}

	dest := strings.TrimSpace(filter.Dest)
	if dest != "" && !strings.EqualFold(dest, "all") {
		whereClauses = append(whereClauses, "TRIM(dest) = ?")
		args = append(args, dest)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	query := `
		SELECT
			id,
			TRIM(no_tiket) AS no_tiket,
			TRIM(reff) AS reff,
			TRIM(nopol) AS nopol,
			TRIM(dk) AS dk,
			TRIM(prd) AS prd,
			TRIM(src) AS src,
			TRIM(dest) AS dest,
			wsrc,
			wdest,
			CASE WHEN wdest > 0 THEN ROUND(wsrc - wdest, 2) ELSE 0 END AS selisih,
			DATE_FORMAT(tgl, '%Y-%m-%d %H:%i:%s') AS tgl,
			IFNULL(DATE_FORMAT(updated, '%Y-%m-%d %H:%i:%s'), '') AS updated
		FROM api_stockpile
		WHERE ` + whereSQL + `
		ORDER BY tgl DESC, id DESC
	`

	var items []models.TransferOutItem
	if err := r.db.Raw(query, args...).Scan(&items).Error; err != nil {
		return nil, err
	}
	if items == nil {
		items = []models.TransferOutItem{}
	}

	// Build summary from filtered items
	destSet := make(map[string]bool)
	prdMap := make(map[string]*models.TransferOutProductSummary)
	destMap := make(map[string]*models.TransferOutDestSummary)
	unitMap := make(map[string]*models.TransferOutUnitSummary)

	var totalWsrc, totalWdest, totalWsrcWithPlant float64
	var totalRitPlant int

	for i := range items {
		it := &items[i]
		it.Wsrc = round2(it.Wsrc)
		it.Wdest = round2(it.Wdest)
		it.Selisih = round2(it.Selisih)

		totalWsrc += it.Wsrc
		totalWdest += it.Wdest
		if it.Wdest > 0 {
			totalRitPlant++
			totalWsrcWithPlant += it.Wsrc
		}

		if it.Nopol != "" && it.Nopol != "0" {
			if _, ok := unitMap[it.Nopol]; !ok {
				unitMap[it.Nopol] = &models.TransferOutUnitSummary{Nopol: it.Nopol}
			}
			unitMap[it.Nopol].JmlRit++
			unitMap[it.Nopol].TotalWsrc += it.Wsrc
			unitMap[it.Nopol].TotalWdest += it.Wdest
		}

		destLabel := it.Dest
		if destLabel == "" || destLabel == "0" {
			destLabel = "(Tanpa Destination)"
		} else {
			destSet[destLabel] = true
		}

		prdLabel := it.Prd
		if prdLabel == "" || prdLabel == "0" {
			prdLabel = "(Tanpa Produk)"
		}

		if _, ok := prdMap[prdLabel]; !ok {
			prdMap[prdLabel] = &models.TransferOutProductSummary{Prd: prdLabel}
		}
		prdMap[prdLabel].JmlRit++
		prdMap[prdLabel].TotalWsrc += it.Wsrc
		prdMap[prdLabel].TotalWdest += it.Wdest

		if _, ok := destMap[destLabel]; !ok {
			destMap[destLabel] = &models.TransferOutDestSummary{Dest: destLabel}
		}
		destMap[destLabel].JmlRit++
		destMap[destLabel].TotalWsrc += it.Wsrc
		destMap[destLabel].TotalWdest += it.Wdest
	}

	byProduct := make([]models.TransferOutProductSummary, 0, len(prdMap))
	for _, v := range prdMap {
		v.TotalWsrc = round2(v.TotalWsrc)
		v.TotalWdest = round2(v.TotalWdest)
		byProduct = append(byProduct, *v)
	}
	sort.Slice(byProduct, func(i, j int) bool {
		return byProduct[i].TotalWsrc > byProduct[j].TotalWsrc
	})

	byDest := make([]models.TransferOutDestSummary, 0, len(destMap))
	for _, v := range destMap {
		v.TotalWsrc = round2(v.TotalWsrc)
		v.TotalWdest = round2(v.TotalWdest)
		byDest = append(byDest, *v)
	}
	sort.Slice(byDest, func(i, j int) bool {
		return byDest[i].TotalWsrc > byDest[j].TotalWsrc
	})

	byUnit := make([]models.TransferOutUnitSummary, 0, len(unitMap))
	for _, v := range unitMap {
		v.TotalWsrc = round2(v.TotalWsrc)
		v.TotalWdest = round2(v.TotalWdest)
		byUnit = append(byUnit, *v)
	}
	sort.Slice(byUnit, func(i, j int) bool {
		if byUnit[i].JmlRit != byUnit[j].JmlRit {
			return byUnit[i].JmlRit > byUnit[j].JmlRit
		}
		if byUnit[i].TotalWsrc != byUnit[j].TotalWsrc {
			return byUnit[i].TotalWsrc > byUnit[j].TotalWsrc
		}
		return byUnit[i].Nopol < byUnit[j].Nopol
	})

	totalRit := len(items)
	var avgWsrcPerRit, selisihPlant, wastePercent float64
	if totalRit > 0 {
		avgWsrcPerRit = round2(totalWsrc / float64(totalRit))
	}
	if totalWsrcWithPlant > 0 {
		selisihPlant = round2(totalWsrcWithPlant - totalWdest)
		wastePercent = round2((selisihPlant / totalWsrcWithPlant) * 100)
	}

	var lastUpdate string
	_ = r.db.Raw(`
		SELECT IFNULL(DATE_FORMAT(MAX(COALESCE(updated, created)), '%d-%b-%Y %H:%i:%s'), '-')
		FROM api_stockpile
	`).Scan(&lastUpdate).Error

	return &models.TransferOutResponse{
		TglAwal:    filter.TglAwal,
		TglAkhir:   filter.TglAkhir,
		LastUpdate: lastUpdate,
		Summary: models.TransferOutSummary{
			TotalRit:           totalRit,
			TotalUnit:          len(byUnit),
			TotalDest:          len(destSet),
			TotalWsrc:          round2(totalWsrc),
			TotalWdest:         round2(totalWdest),
			TotalRitPlant:      totalRitPlant,
			TotalWsrcWithPlant: round2(totalWsrcWithPlant),
			SelisihPlant:       selisihPlant,
			WastePercent:       wastePercent,
			AvgWsrcPerRit:      avgWsrcPerRit,
			ByProduct:          byProduct,
			ByDest:             byDest,
			ByUnit:             byUnit,
		},
		Items: items,
	}, nil
}
