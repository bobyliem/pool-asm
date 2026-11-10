package repository

import (
	"asm/models"
	"sort"
	"strings"

	"gorm.io/gorm"
)

type CsvSpkRepository interface {
	GetFilterOptions() (*models.CsvSpkFilterOptions, error)
	GetSpkPerCust(area, customer, purpose, tglAwal, tglAkhir string) ([]models.CsvSpkPerCustItem, error)
	GetSpkDetail(nopol, tglAwal, tglAkhir string) (*models.CsvSpkDetailResponse, error)
	GetSummaryPerCust(area, customer, purpose, tglAwal, tglAkhir string) (*models.CsvSummaryPerCustResponse, error)
	GetResumePerKategori(area, tglAwal, tglAkhir string) (*models.CsvResumePerKategoriResponse, error)
	GetResumeKategoriDetail(area, customer, category, status, tglAwal, tglAkhir string) ([]models.CsvResumeKategoriDetailItem, error)
}

type csvSpkRepository struct {
	db *gorm.DB
}

func NewCsvSpkRepository(db *gorm.DB) CsvSpkRepository {
	return &csvSpkRepository{db: db}
}

func (r *csvSpkRepository) GetFilterOptions() (*models.CsvSpkFilterOptions, error) {
	var areas []string
	var customers []string
	var purposes []string

	// 1. Query Parameter Area
	errArea := r.db.Raw(`SELECT DISTINCT(site) FROM imp_service_order so WHERE site IS NOT NULL AND site != '' ORDER BY site`).Scan(&areas).Error
	if errArea != nil {
		return nil, errArea
	}

	// 2. Query Parameter Customer
	errCust := r.db.Raw(`SELECT DISTINCT(customer_code) FROM imp_service_order so WHERE customer_code IS NOT NULL AND customer_code != '' ORDER BY customer_code`).Scan(&customers).Error
	if errCust != nil {
		return nil, errCust
	}

	// 3. Query Parameter Purpose
	errPurp := r.db.Raw(`SELECT DISTINCT(category) FROM imp_service_order so WHERE category IS NOT NULL AND category != '' ORDER BY category`).Scan(&purposes).Error
	if errPurp != nil {
		return nil, errPurp
	}

	// 4. Query Last Update Data
	var lastUpdate string
	_ = r.db.Raw(`SELECT DATE_FORMAT(MAX(finished), '%d-%b-%Y %H:%i:%s') AS last_update FROM log_imp`).Scan(&lastUpdate).Error

	return &models.CsvSpkFilterOptions{
		Areas:      areas,
		Customers:  customers,
		Purposes:   purposes,
		LastUpdate: lastUpdate,
	}, nil
}

func (r *csvSpkRepository) GetSpkPerCust(area, customer, purpose, tglAwal, tglAkhir string) ([]models.CsvSpkPerCustItem, error) {
	var items []models.CsvSpkPerCustItem

	query := `
		SELECT MAX(site) AS site, MIN(mulai) AS tgl_masuk, MAX(selesai) AS close_spk
			, MAX(raw_close_date) AS close_date
			, MAX(nopol) AS nopol, MAX(cabin) AS cabin, MAX(cust_code) AS cust_code, MAX(category) AS kategori
			, spk, so_id
			, SUM(service) AS servis, SUM(part) AS part, SUM(ban) AS ban, SUM(total) AS total
			, MAX(last_notes) AS catatan
		FROM(
			SELECT so.service_order AS spk, so.customer_code AS cust_code, so.category
				, so.plate_number AS nopol, so.body_number AS cabin
				, DATE_FORMAT(so.released_at, '%d-%b-%y %H:%i') AS mulai
				, IF(DATE(so.completed_at) IS NULL, NULL, DATE_FORMAT(so.completed_at, '%d-%b-%y %H:%i')) AS selesai
				, DATE_FORMAT(so.completed_at, '%Y-%m-%d') AS raw_close_date
				, CASE WHEN so.tipe = 'Services' THEN so.total_price ELSE 0 END AS service
				, CASE WHEN so.tipe = 'Parts' THEN so.total_price ELSE 0 END AS part
				, CASE WHEN so.tipe = 'Inventories' THEN so.total_price ELSE 0 END AS ban
				, so.total_price AS total
				, site, invoice, so.last_notes, so.visitor_id, so.service_order_id AS so_id
			FROM imp_service_order so
			WHERE 1 
				AND so.statue <> 'Canceled'
				AND (? = '' OR ? = 'ALL' OR so.site = ?)
				AND (? = '' OR ? = 'ALL' OR so.customer_code = ?)
				AND (? = '' OR ? = 'ALL' OR so.category = ?)
				AND (
					DATE(so.released_at) BETWEEN ? AND ?
					OR (DATE(so.completed_at) IS NULL AND DATE(so.released_at) <= ?)
					OR DATE(so.completed_at) BETWEEN ? AND ?
				)
			ORDER BY so.released_at
		) AS dt
		GROUP BY visitor_id, spk, so_id
		ORDER BY MIN(selesai), MAX(mulai)
	`

	err := r.db.Raw(query,
		area, area, area,
		customer, customer, customer,
		purpose, purpose, purpose,
		tglAwal, tglAkhir,
		tglAkhir,
		tglAwal, tglAkhir,
	).Scan(&items).Error

	return items, err
}

func (r *csvSpkRepository) GetSpkDetail(nopol, tglAwal, tglAkhir string) (*models.CsvSpkDetailResponse, error) {
	var services []models.CsvSpkDetailItem
	var spareparts []models.CsvSpkDetailItem

	// 1. Query SERVICE
	queryService := `
		SELECT sku, product, qty, unit_price, total_price, service_order AS spk, category, statue
			, IFNULL(DATE_FORMAT(completed_at, '%d-%b-%Y %H:%i'), '-') AS selesai
			, IFNULL(DATE_FORMAT(released_at, '%d-%b-%Y %H:%i'), '-') AS tgl_masuk
			, DATE_FORMAT(completed_at, '%Y-%m-%d') AS raw_close_date
		FROM imp_service_order so
		WHERE 1
			AND so.plate_number = ?
			AND so.tipe = 'Services'
			AND (
				DATE(so.released_at) BETWEEN ? AND ?
				OR (DATE(so.completed_at) IS NULL AND DATE(so.released_at) <= ?)
				OR DATE(so.completed_at) BETWEEN ? AND ?
			)
		ORDER BY so.service_order, so.id
	`
	if err := r.db.Raw(queryService, nopol, tglAwal, tglAkhir, tglAkhir, tglAwal, tglAkhir).Scan(&services).Error; err != nil {
		return nil, err
	}

	// 2. Query SPAREPART
	querySparepart := `
		SELECT sku, product, qty, unit_price, total_price, service_order AS spk, category, statue
			, IFNULL(DATE_FORMAT(completed_at, '%d-%b-%Y %H:%i'), '-') AS selesai
			, IFNULL(DATE_FORMAT(released_at, '%d-%b-%Y %H:%i'), '-') AS tgl_masuk
			, DATE_FORMAT(completed_at, '%Y-%m-%d') AS raw_close_date
		FROM imp_service_order so
		WHERE 1
			AND so.plate_number = ?
			AND so.tipe <> 'Services'
			AND (
				DATE(so.released_at) BETWEEN ? AND ?
				OR (DATE(so.completed_at) IS NULL AND DATE(so.released_at) <= ?)
				OR DATE(so.completed_at) BETWEEN ? AND ?
			)
		ORDER BY so.service_order, so.id
	`
	if err := r.db.Raw(querySparepart, nopol, tglAwal, tglAkhir, tglAkhir, tglAwal, tglAkhir).Scan(&spareparts).Error; err != nil {
		return nil, err
	}

	// 3. Query NOTES per SPK yang memiliki isi
	queryNotes := `
		SELECT service_order AS spk, MAX(last_notes) AS notes
		FROM imp_service_order so
		WHERE 1
			AND so.plate_number = ?
			AND (
				DATE(so.released_at) BETWEEN ? AND ?
				OR (DATE(so.completed_at) IS NULL AND DATE(so.released_at) <= ?)
				OR DATE(so.completed_at) BETWEEN ? AND ?
			)
			AND so.last_notes IS NOT NULL
			AND TRIM(so.last_notes) <> ''
			AND TRIM(so.last_notes) <> '-'
		GROUP BY service_order
		ORDER BY service_order
	`
	var notes []models.CsvSpkNoteItem
	_ = r.db.Raw(queryNotes, nopol, tglAwal, tglAkhir, tglAkhir, tglAwal, tglAkhir).Scan(&notes).Error

	// 4. Kalkulasi Status SPK (CLOSE, HOLD, OPEN)
	determineStatus := func(rawCloseDate *string) string {
		if rawCloseDate == nil || *rawCloseDate == "" || *rawCloseDate == "-" {
			return "OPEN"
		}
		if *rawCloseDate > tglAkhir {
			return "HOLD"
		}
		return "CLOSE"
	}

	spkStatusMap := make(map[string]string)
	for _, item := range services {
		if item.Spk != "" {
			if _, ok := spkStatusMap[item.Spk]; !ok {
				spkStatusMap[item.Spk] = determineStatus(item.RawCloseDate)
			}
		}
	}
	for _, item := range spareparts {
		if item.Spk != "" {
			if _, ok := spkStatusMap[item.Spk]; !ok {
				spkStatusMap[item.Spk] = determineStatus(item.RawCloseDate)
			}
		}
	}

	for i := range services {
		if st, ok := spkStatusMap[services[i].Spk]; ok {
			services[i].SpkStatus = st
		} else {
			services[i].SpkStatus = determineStatus(services[i].RawCloseDate)
		}
	}
	for i := range spareparts {
		if st, ok := spkStatusMap[spareparts[i].Spk]; ok {
			spareparts[i].SpkStatus = st
		} else {
			spareparts[i].SpkStatus = determineStatus(spareparts[i].RawCloseDate)
		}
	}

	// 5. Kalkulasi Ringkasan Summary
	var totalService float64
	var totalSparepart float64
	var earliestMasuk string

	var closedMetrics, holdMetrics, openMetrics models.CsvSpkDetailMetrics

	for _, st := range spkStatusMap {
		switch st {
		case "CLOSE":
			closedMetrics.SpkCount++
		case "HOLD":
			holdMetrics.SpkCount++
		case "OPEN":
			openMetrics.SpkCount++
		}
	}

	for _, item := range services {
		if !strings.EqualFold(item.Statue, "Canceled") {
			totalService += item.TotalPrice
			switch item.SpkStatus {
			case "CLOSE":
				closedMetrics.Service += item.TotalPrice
				closedMetrics.Total += item.TotalPrice
			case "HOLD":
				holdMetrics.Service += item.TotalPrice
				holdMetrics.Total += item.TotalPrice
			case "OPEN":
				openMetrics.Service += item.TotalPrice
				openMetrics.Total += item.TotalPrice
			}
		}
		if item.TglMasuk != "" && item.TglMasuk != "-" {
			if earliestMasuk == "" || item.TglMasuk < earliestMasuk {
				earliestMasuk = item.TglMasuk
			}
		}
	}

	for _, item := range spareparts {
		if !strings.EqualFold(item.Statue, "Canceled") {
			totalSparepart += item.TotalPrice
			switch item.SpkStatus {
			case "CLOSE":
				closedMetrics.Part += item.TotalPrice
				closedMetrics.Total += item.TotalPrice
			case "HOLD":
				holdMetrics.Part += item.TotalPrice
				holdMetrics.Total += item.TotalPrice
			case "OPEN":
				openMetrics.Part += item.TotalPrice
				openMetrics.Total += item.TotalPrice
			}
		}
		if item.TglMasuk != "" && item.TglMasuk != "-" {
			if earliestMasuk == "" || item.TglMasuk < earliestMasuk {
				earliestMasuk = item.TglMasuk
			}
		}
	}

	res := &models.CsvSpkDetailResponse{
		Nopol:            nopol,
		TglAwal:          tglAwal,
		TglAkhir:         tglAkhir,
		TglMasuk:         earliestMasuk,
		TotalSpk:         len(spkStatusMap),
		TotalService:     totalService,
		TotalSparepart:   totalSparepart,
		TotalKeseluruhan: totalService + totalSparepart,
		Closed:           closedMetrics,
		Hold:             holdMetrics,
		Open:             openMetrics,
		Notes:            notes,
		Services:         services,
		Spareparts:       spareparts,
	}

	return res, nil
}

func (r *csvSpkRepository) GetSummaryPerCust(area, customer, purpose, tglAwal, tglAkhir string) (*models.CsvSummaryPerCustResponse, error) {
	items, err := r.GetSpkPerCust(area, customer, purpose, tglAwal, tglAkhir)
	if err != nil {
		return nil, err
	}

	type catHolder struct {
		category string
		total    models.CsvSpkSummaryMetrics
		closed   models.CsvSpkSummaryMetrics
		open     models.CsvSpkSummaryMetrics
		hold     models.CsvSpkSummaryMetrics
	}

	type custHolder struct {
		customer string
		catMap   map[string]*catHolder
		catOrder []string
		total    models.CsvSpkSummaryMetrics
		closed   models.CsvSpkSummaryMetrics
		open     models.CsvSpkSummaryMetrics
		hold     models.CsvSpkSummaryMetrics
	}

	custMap := make(map[string]*custHolder)
	var custOrder []string
	var grandTotal models.CsvGrandTotalSummary

	for _, item := range items {
		cust := strings.TrimSpace(item.CustCode)
		if cust == "" {
			cust = "UNKNOWN"
		}
		cat := strings.TrimSpace(item.Kategori)
		if cat == "" {
			cat = "Others"
		}

		cHolder, ok := custMap[cust]
		if !ok {
			cHolder = &custHolder{
				customer: cust,
				catMap:   make(map[string]*catHolder),
				catOrder: make([]string, 0),
			}
			custMap[cust] = cHolder
			custOrder = append(custOrder, cust)
		}

		kHolder, kOk := cHolder.catMap[cat]
		if !kOk {
			kHolder = &catHolder{
				category: cat,
			}
			cHolder.catMap[cat] = kHolder
			cHolder.catOrder = append(cHolder.catOrder, cat)
		}

		isClosed := item.CloseSpk != nil && *item.CloseSpk != "" && *item.CloseSpk != "-"
		isHold := false
		if isClosed && item.CloseDate != nil && *item.CloseDate != "" {
			if *item.CloseDate > tglAkhir {
				isHold = true
			}
		}

		// 1. Add to Category
		kHolder.total.SpkCount++
		kHolder.total.TotalJasa += item.Servis
		kHolder.total.TotalPart += item.Part
		kHolder.total.TotalBan += item.Ban
		kHolder.total.GrandTotal += item.Total

		if isHold {
			kHolder.hold.SpkCount++
			kHolder.hold.TotalJasa += item.Servis
			kHolder.hold.TotalPart += item.Part
			kHolder.hold.TotalBan += item.Ban
			kHolder.hold.GrandTotal += item.Total
		} else if isClosed {
			kHolder.closed.SpkCount++
			kHolder.closed.TotalJasa += item.Servis
			kHolder.closed.TotalPart += item.Part
			kHolder.closed.TotalBan += item.Ban
			kHolder.closed.GrandTotal += item.Total
		} else {
			kHolder.open.SpkCount++
			kHolder.open.TotalJasa += item.Servis
			kHolder.open.TotalPart += item.Part
			kHolder.open.TotalBan += item.Ban
			kHolder.open.GrandTotal += item.Total
		}

		// 2. Add to Customer Total
		cHolder.total.SpkCount++
		cHolder.total.TotalJasa += item.Servis
		cHolder.total.TotalPart += item.Part
		cHolder.total.TotalBan += item.Ban
		cHolder.total.GrandTotal += item.Total

		if isHold {
			cHolder.hold.SpkCount++
			cHolder.hold.TotalJasa += item.Servis
			cHolder.hold.TotalPart += item.Part
			cHolder.hold.TotalBan += item.Ban
			cHolder.hold.GrandTotal += item.Total
		} else if isClosed {
			cHolder.closed.SpkCount++
			cHolder.closed.TotalJasa += item.Servis
			cHolder.closed.TotalPart += item.Part
			cHolder.closed.TotalBan += item.Ban
			cHolder.closed.GrandTotal += item.Total
		} else {
			cHolder.open.SpkCount++
			cHolder.open.TotalJasa += item.Servis
			cHolder.open.TotalPart += item.Part
			cHolder.open.TotalBan += item.Ban
			cHolder.open.GrandTotal += item.Total
		}

		// 3. Add to Grand Total
		grandTotal.Total.SpkCount++
		grandTotal.Total.TotalJasa += item.Servis
		grandTotal.Total.TotalPart += item.Part
		grandTotal.Total.TotalBan += item.Ban
		grandTotal.Total.GrandTotal += item.Total

		if isHold {
			grandTotal.Hold.SpkCount++
			grandTotal.Hold.TotalJasa += item.Servis
			grandTotal.Hold.TotalPart += item.Part
			grandTotal.Hold.TotalBan += item.Ban
			grandTotal.Hold.GrandTotal += item.Total
		} else if isClosed {
			grandTotal.Closed.SpkCount++
			grandTotal.Closed.TotalJasa += item.Servis
			grandTotal.Closed.TotalPart += item.Part
			grandTotal.Closed.TotalBan += item.Ban
			grandTotal.Closed.GrandTotal += item.Total
		} else {
			grandTotal.Open.SpkCount++
			grandTotal.Open.TotalJasa += item.Servis
			grandTotal.Open.TotalPart += item.Part
			grandTotal.Open.TotalBan += item.Ban
			grandTotal.Open.GrandTotal += item.Total
		}
	}

	sort.Strings(custOrder)

	customers := make([]models.CsvCustomerSummary, 0, len(custOrder))
	for _, custName := range custOrder {
		cHolder := custMap[custName]
		sort.Strings(cHolder.catOrder)

		cats := make([]models.CsvCategorySummary, 0, len(cHolder.catOrder))
		for _, catName := range cHolder.catOrder {
			kHolder := cHolder.catMap[catName]
			cats = append(cats, models.CsvCategorySummary{
				Category: kHolder.category,
				Total:    kHolder.total,
				Closed:   kHolder.closed,
				Open:     kHolder.open,
				Hold:     kHolder.hold,
			})
		}

		customers = append(customers, models.CsvCustomerSummary{
			Customer:   cHolder.customer,
			Categories: cats,
			Total:      cHolder.total,
			Closed:     cHolder.closed,
			Open:       cHolder.open,
			Hold:       cHolder.hold,
		})
	}

	return &models.CsvSummaryPerCustResponse{
		TglAwal:    tglAwal,
		TglAkhir:   tglAkhir,
		Customers:  customers,
		GrandTotal: grandTotal,
	}, nil
}

func (r *csvSpkRepository) GetResumePerKategori(area, tglAwal, tglAkhir string) (*models.CsvResumePerKategoriResponse, error) {
	items, err := r.GetSpkPerCust(area, "ALL", "ALL", tglAwal, tglAkhir)
	if err != nil {
		return nil, err
	}

	custSet := make(map[string]bool)
	catSet := make(map[string]bool)

	for _, item := range items {
		cust := strings.TrimSpace(item.CustCode)
		if cust == "" {
			cust = "UNKNOWN"
		}
		custSet[cust] = true

		cat := strings.TrimSpace(item.Kategori)
		if cat == "" {
			cat = "Others"
		}
		catSet[cat] = true
	}

	prefCats := []string{"Perbaikan", "Asuransi", "Storing", "Penjualan", "Other", "Others"}
	var categories []string
	addedCats := make(map[string]bool)

	for _, pc := range prefCats {
		if catSet[pc] && !addedCats[pc] {
			categories = append(categories, pc)
			addedCats[pc] = true
		}
	}

	var remainingCats []string
	for cat := range catSet {
		if !addedCats[cat] {
			remainingCats = append(remainingCats, cat)
		}
	}
	sort.Strings(remainingCats)
	categories = append(categories, remainingCats...)

	var customers []string
	for cust := range custSet {
		customers = append(customers, cust)
	}
	sort.Strings(customers)

	buildSection := func(title string, filterFunc func(item models.CsvSpkPerCustItem) bool) models.CsvResumeMatrixSection {
		matrix := make(map[string]map[string]*models.CsvResumeMatrixCell)
		for _, cust := range customers {
			matrix[cust] = make(map[string]*models.CsvResumeMatrixCell)
			for _, cat := range categories {
				matrix[cust][cat] = &models.CsvResumeMatrixCell{}
			}
		}

		for _, item := range items {
			if !filterFunc(item) {
				continue
			}
			cust := strings.TrimSpace(item.CustCode)
			if cust == "" {
				cust = "UNKNOWN"
			}
			cat := strings.TrimSpace(item.Kategori)
			if cat == "" {
				cat = "Others"
			}

			if cellMap, ok := matrix[cust]; ok {
				if cell, cOk := cellMap[cat]; cOk {
					cell.Amount += item.Total
					cell.SpkCount++
				}
			}
		}

		rows := make([]models.CsvResumeCustomerRow, 0, len(customers))
		catTotals := make(map[string]models.CsvResumeMatrixCell)
		for _, cat := range categories {
			catTotals[cat] = models.CsvResumeMatrixCell{}
		}
		var grandTotal models.CsvResumeMatrixCell

		for _, cust := range customers {
			rowValues := make(map[string]models.CsvResumeMatrixCell)
			var rowTotal models.CsvResumeMatrixCell

			for _, cat := range categories {
				cell := *matrix[cust][cat]
				rowValues[cat] = cell
				rowTotal.Amount += cell.Amount
				rowTotal.SpkCount += cell.SpkCount

				ct := catTotals[cat]
				ct.Amount += cell.Amount
				ct.SpkCount += cell.SpkCount
				catTotals[cat] = ct
			}

			grandTotal.Amount += rowTotal.Amount
			grandTotal.SpkCount += rowTotal.SpkCount

			rows = append(rows, models.CsvResumeCustomerRow{
				Customer: cust,
				Values:   rowValues,
				Total:    rowTotal,
			})
		}

		totalRow := models.CsvResumeCustomerRow{
			Customer: "Total",
			Values:   catTotals,
			Total:    grandTotal,
		}

		return models.CsvResumeMatrixSection{
			Title:    title,
			Rows:     rows,
			TotalRow: totalRow,
		}
	}

	isClosedOnly := func(item models.CsvSpkPerCustItem) bool {
		isClosed := item.CloseSpk != nil && *item.CloseSpk != "" && *item.CloseSpk != "-"
		if !isClosed {
			return false
		}
		if item.CloseDate != nil && *item.CloseDate != "" && *item.CloseDate > tglAkhir {
			return false
		}
		return true
	}
	isHold := func(item models.CsvSpkPerCustItem) bool {
		isClosed := item.CloseSpk != nil && *item.CloseSpk != "" && *item.CloseSpk != "-"
		if !isClosed {
			return false
		}
		return item.CloseDate != nil && *item.CloseDate != "" && *item.CloseDate > tglAkhir
	}
	isOpen := func(item models.CsvSpkPerCustItem) bool {
		isClosed := item.CloseSpk != nil && *item.CloseSpk != "" && *item.CloseSpk != "-"
		return !isClosed
	}
	isAll := func(item models.CsvSpkPerCustItem) bool {
		return true
	}

	totalSection := buildSection("Total SPK", isAll)
	closedSection := buildSection("SPK Close", isClosedOnly)
	holdSection := buildSection("SPK HOLD", isHold)
	openSection := buildSection("SPK Open", isOpen)

	grandTotalMatrix := make(map[string]map[string]models.CsvResumeMatrixCell)
	grandTotalSummary := make(map[string]models.CsvResumeMatrixCell)

	grandTotalMatrix["SPK Open"] = openSection.TotalRow.Values
	grandTotalSummary["SPK Open"] = openSection.TotalRow.Total

	grandTotalMatrix["SPK HOLD"] = holdSection.TotalRow.Values
	grandTotalSummary["SPK HOLD"] = holdSection.TotalRow.Total

	grandTotalMatrix["SPK Close"] = closedSection.TotalRow.Values
	grandTotalSummary["SPK Close"] = closedSection.TotalRow.Total

	grandTotalMatrix["Total SPK"] = totalSection.TotalRow.Values
	grandTotalSummary["Total SPK"] = totalSection.TotalRow.Total

	return &models.CsvResumePerKategoriResponse{
		TglAwal:           tglAwal,
		TglAkhir:          tglAkhir,
		Categories:        categories,
		GrandTotalMatrix:  grandTotalMatrix,
		GrandTotalSummary: grandTotalSummary,
		TotalSpkSection:   totalSection,
		SpkCloseSection:   closedSection,
		SpkHoldSection:    holdSection,
		SpkOpenSection:    openSection,
	}, nil
}

func (r *csvSpkRepository) GetResumeKategoriDetail(area, customer, category, status, tglAwal, tglAkhir string) ([]models.CsvResumeKategoriDetailItem, error) {
	var items []models.CsvResumeKategoriDetailItem

	query := `
		SELECT nopol, COUNT(so) AS spk, cat AS kategori
			, FLOOR(SUM(service)) AS service
			, FLOOR(SUM(part)) AS part
			, FLOOR(SUM(ban)) AS ban
			, FLOOR(SUM(total)) AS total
			, MAX(note) AS note
		FROM (
			SELECT nopol, so, MAX(category) AS cat
				, FLOOR(SUM(service)) AS service
				, FLOOR(SUM(part)) AS part
				, FLOOR(SUM(ban)) AS ban
				, FLOOR(SUM(total)) AS total
				, MAX(note) AS note
			FROM (
				SELECT service_order AS so, customer_code AS cust, plate_number AS nopol
					, CASE WHEN tipe = 'Services' THEN total_price ELSE 0 END AS service
					, CASE WHEN tipe = 'Parts' THEN total_price ELSE 0 END AS part
					, CASE WHEN tipe = 'Inventories' THEN total_price ELSE 0 END AS ban
					, total_price AS total, category
					, last_notes AS note
					, DATE(completed_at)
				FROM imp_service_order so
				WHERE 1
					AND so.statue <> 'Canceled'
					AND (? = '' OR ? = 'ALL' OR so.site = ?)
					AND (
						DATE(so.released_at) BETWEEN ? AND ?
						OR (DATE(so.completed_at) IS NULL AND DATE(so.released_at) <= ?)
						OR DATE(so.completed_at) BETWEEN ? AND ?
					)
					AND (? = '' OR ? = 'ALL' OR so.customer_code = ?)
					AND (? = '' OR ? = 'ALL' OR so.category = ?)
	`

	args := []interface{}{
		area, area, area,
		tglAwal, tglAkhir,
		tglAkhir,
		tglAwal, tglAkhir,
		customer, customer, customer,
		category, category, category,
	}

	upperStatus := strings.ToUpper(strings.TrimSpace(status))
	if upperStatus == "OPEN" {
		query += ` AND DATE(so.completed_at) IS NULL `
	} else if upperStatus == "CLOSE" || upperStatus == "CLOSED" {
		query += ` AND DATE(so.completed_at) IS NOT NULL AND DATE(so.completed_at) <= ? `
		args = append(args, tglAkhir)
	} else if upperStatus == "HOLD" {
		query += ` AND DATE(so.completed_at) IS NOT NULL AND DATE(so.completed_at) > ? `
		args = append(args, tglAkhir)
	}

	query += `
				) AS dt
				GROUP BY nopol, so
				ORDER BY total DESC
			) AS dt
			GROUP BY nopol, kategori
			ORDER BY kategori, nopol, total DESC
	`

	err := r.db.Raw(query, args...).Scan(&items).Error
	return items, err
}
