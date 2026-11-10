package repository

import (
	"asm/models"
	"gorm.io/gorm"
)

type OperationRepository interface {
	GetOperationSummary() (*models.OperationSummary, error)
}

type operationRepository struct {
	db *gorm.DB
}

func NewOperationRepository(db *gorm.DB) OperationRepository {
	return &operationRepository{db: db}
}

func (r *operationRepository) GetOperationSummary() (*models.OperationSummary, error) {
	// Query Konfirmasi Kelengkapan langsung dari MySQL database 'sampurna'
	var kelengkapan models.KelengkapanSummary
	queryKelengkapan := `
		SELECT 
			COALESCE(SUM(asuransi), 0) AS asuransi, 
			COALESCE(SUM(perbaikan), 0) AS perbaikan, 
			COALESCE(SUM(storing), 0) AS storing,
			COALESCE(SUM(other), 0) AS other,
			COALESCE(SUM(total), 0) AS total
		FROM (
			SELECT 1 AS total
				, CASE WHEN s.category_value = 'Asuransi' THEN 1 ELSE 0 END AS asuransi
				, CASE WHEN s.category_value = 'Perbaikan' THEN 1 ELSE 0 END AS perbaikan
				, CASE WHEN s.category_value = 'Storing' THEN 1 ELSE 0 END AS storing
				, CASE WHEN s.category_value <> 'Asuransi' AND s.category_value <> 'Perbaikan' AND s.category_value <> 'Storing' THEN 1 ELSE 0 END AS other
			FROM service_order_confirmations c
			INNER JOIN service_order_inspections i ON c.serviceOrderInspectionId = i.id
			INNER JOIN service_orders s ON i.serviceOrderId = s.id
			INNER JOIN vehicles v ON i.vehicleId = v.id
			WHERE c.deletedAt IS NULL
				AND s.deletedAt IS NULL
				AND c.status = 1
				AND s.status != 10
				AND s.siteId = '00000017-b43e-11ee-bde1-bfcb6ff3c685'
		) AS dt
	`

	if err := r.db.Raw(queryKelengkapan).Scan(&kelengkapan).Error; err != nil {
		kelengkapan = models.KelengkapanSummary{
			Perbaikan: 0,
			Asuransi:  0,
			Storing:   0,
			Other:     0,
			Total:     0,
		}
	}

	// Query Overdue langsung dari MySQL database 'sampurna'
	var overdue models.OverdueSummary
	queryOverdue := `
		SELECT 
			COALESCE(SUM(perbaikan), 0) AS perbaikan, 
			COALESCE(SUM(asuransi), 0) AS asuransi, 
			COALESCE(SUM(storing), 0) AS storing, 
			COALESCE(SUM(other), 0) AS other, 
			COALESCE(SUM(total), 0) AS total
		FROM (
			SELECT so.category_value
				, CASE WHEN so.category_value = 'Perbaikan' THEN 1 ELSE 0 END AS perbaikan
				, CASE WHEN so.category_value = 'Asuransi' THEN 1 ELSE 0 END AS asuransi
				, CASE WHEN so.category_value = 'Storing' THEN 1 ELSE 0 END AS storing
				, CASE WHEN so.category_value = 'Others' THEN 1 ELSE 0 END AS other
				, 1 AS total
			FROM service_orders so
			LEFT JOIN vehicles v ON so.vehicleId = v.id
			WHERE so.status NOT IN(4, 8, 10)
				AND so.deletedAt IS NULL
				AND so.completedAt IS NULL
				AND DATE(IFNULL(so.releasedAt, '2000-01-01')) < CURDATE()
				AND so.siteId = '00000017-b43e-11ee-bde1-bfcb6ff3c685'
		) AS dt
	`

	if err := r.db.Raw(queryOverdue).Scan(&overdue).Error; err != nil {
		overdue = models.OverdueSummary{
			Perbaikan: 0,
			Asuransi:  0,
			Storing:   0,
			Other:     0,
			Total:     0,
		}
	}

	// Query 6 Kolom Kategori (Visitors Status In/Out/Saldo)
	columnStats := make(map[string]models.ColumnCategoryStat)
	var statsList []models.ColumnCategoryStat
	queryColumnStats := `
		SELECT kategori
			, COALESCE(SUM(last_in), 0) AS last_in
			, COALESCE(SUM(last_out), 0) AS last_out
			, COALESCE(SUM(last_in) - SUM(last_out), 0) AS last_saldo
			, COALESCE(SUM(now_in), 0) AS now_in
			, COALESCE(SUM(now_out), 0) AS now_out 
			, COALESCE(SUM(now_in) - SUM(now_out), 0) AS now_saldo
			, COALESCE(SUM(last_in) - SUM(last_out) + SUM(now_in) - SUM(now_out), 0) AS total
		FROM (
			SELECT vs.purpose_value AS kategori
				, CASE WHEN DATE(vs.entryDate) < CURDATE() THEN 1 ELSE 0 END AS last_in
				, CASE WHEN DATE(vs.entryDate) < CURDATE() AND DATE(vs.exitDate) = CURDATE() THEN 1 ELSE 0 END AS last_out
				, CASE WHEN DATE(vs.entryDate) = CURDATE() THEN 1 ELSE 0 END AS now_in
				, CASE WHEN DATE(vs.entryDate) = CURDATE() AND DATE(vs.exitDate) = CURDATE() THEN 1 ELSE 0 END AS now_out
			FROM visitors vs
			WHERE vs.deletedAt IS NULL
				AND vs.siteId = '00000017-b43e-11ee-bde1-bfcb6ff3c685'
				AND (DATE(vs.entryDate) = CURDATE() OR vs.exitDate IS NULL OR DATE(vs.exitDate) = CURDATE())
		) AS dt
		GROUP BY kategori
	`

	if err := r.db.Raw(queryColumnStats).Scan(&statsList).Error; err == nil {
		for _, stat := range statsList {
			columnStats[stat.Kategori] = stat
		}
	}

	// Query List Kartu Unit per Kategori
	unitCardsMap := make(map[string][]models.UnitCardItem)

	// 1. Query Kartu Unit Kategori Umum (Perbaikan, Storing, Standby, Bongkar Muat, Tunggu DO)
	var defaultUnitCards []models.UnitCardItem
	queryDefaultCards := `
		SELECT IFNULL(est.statue, '-') AS statue
			, IFNULL(DATE_FORMAT(vs.entryDate, '%d %b %Y %H:%i'), '-') AS entryDate
			, vh.category_value AS kategori
			, vs.purpose_value AS purpose_kategori
			, IFNULL(vsd.name, '-') AS driver
			, CONCAT(vh.plate_number, ' [', IFNULL(vh.type_value, ''), ']') AS nopol
			, c.code AS CustCode
			, IFNULL(est.estimasi, '-') AS estimasi
			, IFNULL(est.is_overdue, 0) AS is_overdue
		FROM visitors vs
		INNER JOIN vehicles vh ON vs.vehicleId = vh.id
		INNER JOIN customers c ON vs.customerId = c.id
		INNER JOIN sites s ON vs.siteId = s.id
		LEFT JOIN drivers d ON vh.driverId = d.id
		LEFT JOIN drivers vsd ON vs.driverId = vsd.id
		LEFT JOIN(
			SELECT vs.id, IFNULL(DATE_FORMAT(so.estimated_completed, '%d %b %Y %H:%i'), '-') AS estimasi
				, CASE WHEN DATE(so.estimated_completed) < CURDATE() THEN 1 ELSE 0 END AS is_overdue
				, CASE
					WHEN so.status BETWEEN 1 AND 4 THEN 'Antrian'
					WHEN so.status = 5 THEN 'Pengerjaan'
					WHEN so.status BETWEEN 6 AND 9 THEN 'Closing'
					WHEN so.status = 10 THEN 'Cancel'
					WHEN so.status = 11 THEN 'TungguSurvey'
					ELSE '-'
				END AS statue
				, so.status AS a
			FROM service_orders so
			LEFT JOIN visitors vs ON so.visitorId = vs.id
			WHERE vs.deletedAt IS NULL
				AND vs.siteId = '00000017-b43e-11ee-bde1-bfcb6ff3c685'
				AND (DATE(vs.entryDate) = CURDATE() OR vs.exitDate IS NULL OR DATE(vs.exitDate) = CURDATE())
		) AS est ON vs.id = est.id
		WHERE vs.deletedAt IS NULL
			AND vs.siteId = '00000017-b43e-11ee-bde1-bfcb6ff3c685'
			AND vs.purpose_value != 'Asuransi'
			AND (DATE(vs.entryDate) = CURDATE() OR vs.exitDate IS NULL OR DATE(vs.exitDate) = CURDATE())
		ORDER BY est.a, vs.exitDate, DATE(vs.entryDate)
	`

	if err := r.db.Raw(queryDefaultCards).Scan(&defaultUnitCards).Error; err == nil {
		for _, item := range defaultUnitCards {
			key := item.PurposeKategori
			if key == "" {
				key = item.Kategori
			}
			unitCardsMap[key] = append(unitCardsMap[key], item)
		}
	}

	// 2. Query Khusus Kartu Unit Kategori Asuransi
	var asuransiUnitCards []models.UnitCardItem
	queryAsuransiCards := `
		SELECT IFNULL(est.statue, '-') AS statue
			, IFNULL(DATE_FORMAT(vs.entryDate, '%d %b %Y %H:%i'), '-') AS entryDate
			, vh.category_value AS kategori
			, vs.purpose_value AS purpose_kategori
			, IFNULL(vsd.name, '-') AS driver
			, CONCAT(vh.plate_number, ' [', IFNULL(vh.type_value, ''), ']') AS nopol
			, c.code AS CustCode
			, IFNULL(est.estimasi, '-') AS estimasi
			, IFNULL(est.is_overdue, 0) AS is_overdue
		FROM visitors vs
		INNER JOIN vehicles vh ON vs.vehicleId = vh.id
		INNER JOIN customers c ON vs.customerId = c.id
		INNER JOIN sites s ON vs.siteId = s.id
		LEFT JOIN drivers d ON vh.driverId = d.id
		LEFT JOIN drivers vsd ON vs.driverId = vsd.id
		LEFT JOIN(
			SELECT vs.id, IFNULL(DATE_FORMAT(so.estimated_completed, '%d %b %Y %H:%i'), '-') AS estimasi
				, CASE WHEN DATE(so.estimated_completed) < CURDATE() THEN 1 ELSE 0 END AS is_overdue
				, CASE
					WHEN so.status BETWEEN 1 AND 4 THEN 'Antrian'
					WHEN so.status = 5 THEN 'Pengerjaan'
					WHEN so.status BETWEEN 6 AND 9 THEN 'Closing'
					WHEN so.status = 10 THEN 'Cancel'
					WHEN so.status = 11 THEN 'TungguSurvey'
					ELSE '-'
				END AS statue
				, so.status AS a
			FROM service_orders so
			LEFT JOIN visitors vs ON so.visitorId = vs.id
			WHERE vs.deletedAt IS NULL
				AND vs.siteId = '00000017-b43e-11ee-bde1-bfcb6ff3c685'
				AND vs.purpose_value = 'Asuransi'
				AND (DATE(vs.entryDate) = CURDATE() OR vs.exitDate IS NULL OR DATE(vs.exitDate) = CURDATE())
				AND so.status NOT IN(8)
		) AS est ON vs.id = est.id
		WHERE vs.deletedAt IS NULL
			AND vs.siteId = '00000017-b43e-11ee-bde1-bfcb6ff3c685'
			AND vs.purpose_value = 'Asuransi'
			AND (DATE(vs.entryDate) = CURDATE() OR vs.exitDate IS NULL OR DATE(vs.exitDate) = CURDATE())
			AND est.a <> 7
		ORDER BY est.a, vs.exitDate, DATE(vs.entryDate)
	`

	if err := r.db.Raw(queryAsuransiCards).Scan(&asuransiUnitCards).Error; err == nil {
		for _, item := range asuransiUnitCards {
			key := item.PurposeKategori
			if key == "" {
				key = item.Kategori
			}
			unitCardsMap[key] = append(unitCardsMap[key], item)
		}
	}

	summary := &models.OperationSummary{
		TotalActiveSO:    18,
		InRepairCount:    10,
		WaitingPartCount: 4,
		CompletedToday:   6,
		StoringEmergency: 2,
		Kelengkapan:      kelengkapan,
		Overdue:          overdue,
		ColumnStats:      columnStats,
		UnitCards:        unitCardsMap,
		ActiveOperations: []models.ActiveOperationItem{
			{
				ID:           1,
				OrderNumber:  "SO-202609-012",
				PlateNumber:  "B 9123 SPU",
				VehicleType:  "Colt Diesel Engkel",
				WorkStatus:   "Sedang Dikerjakan",
				MechanicName: "Budi Santoso",
				Progress:     75,
				EntryDate:    "27 Sep 2026 08:30",
				EstimatedETA: "27 Sep 2026 16:00",
			},
			{
				ID:           2,
				OrderNumber:  "SO-202609-015",
				PlateNumber:  "B 9844 UYT",
				VehicleType:  "Fuso Tronton",
				WorkStatus:   "Tunggu Sparepart",
				MechanicName: "Ahmad Rizki",
				Progress:     40,
				EntryDate:    "27 Sep 2026 09:15",
				EstimatedETA: "28 Sep 2026 11:00",
			},
			{
				ID:           3,
				OrderNumber:  "SO-202609-018",
				PlateNumber:  "B 8211 KLP",
				VehicleType:  "Pickup L300",
				WorkStatus:   "Uji Coba / QC",
				MechanicName: "Dedi Prasetyo",
				Progress:     90,
				EntryDate:    "27 Sep 2026 07:45",
				EstimatedETA: "27 Sep 2026 14:30",
			},
			{
				ID:           4,
				OrderNumber:  "SO-202609-020",
				PlateNumber:  "B 9500 TKS",
				VehicleType:  "Canter HD 125",
				WorkStatus:   "Pemeriksaan Awal",
				MechanicName: "Hendra Wijaya",
				Progress:     15,
				EntryDate:    "27 Sep 2026 10:00",
				EstimatedETA: "27 Sep 2026 17:30",
			},
		},
	}

	return summary, nil
}

