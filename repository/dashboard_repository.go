package repository

import (
	"asm/models"
	"gorm.io/gorm"
)

type CsvDashboardRepository interface {
	GetVisitorSummary() (*models.DashboardVisitorSummary, error)
	GetAreas() ([]string, error)
	GetCategoryDashboardData(area string) (*models.CsvDashboardDataResponse, error)
}

type csvDashboardRepository struct {
	db *gorm.DB
}

func NewCsvDashboardRepository(db *gorm.DB) CsvDashboardRepository {
	return &csvDashboardRepository{db: db}
}

func (r *csvDashboardRepository) GetVisitorSummary() (*models.DashboardVisitorSummary, error) {
	summary := &models.DashboardVisitorSummary{
		TotalUnitInPool:   0,
		PreviousPoolUnit:  0,
		Masuk:             0,
		Keluar:            0,
		StatusBreakdown:   make(map[string]models.StatusDetail),
		VehicleByType:     []models.CategoryCount{},
		VehicleByCategory: []models.CategoryCount{},
	}
	return summary, nil
}

func (r *csvDashboardRepository) GetAreas() ([]string, error) {
	var areas []string
	query := "SELECT DISTINCT(site) FROM imp_service_order WHERE site IS NOT NULL AND site <> '' ORDER BY site"
	err := r.db.Raw(query).Scan(&areas).Error
	return areas, err
}

func (r *csvDashboardRepository) GetCategoryDashboardData(area string) (*models.CsvDashboardDataResponse, error) {
	areas, err := r.GetAreas()
	if err != nil {
		areas = []string{}
	}

	purposes := []string{"Perbaikan", "Asuransi", "Storing", "Standby", "Bongkar Muat", "Tunggu DO"}
	counts := make(map[string]int)
	categories := make(map[string][]models.CsvVisitorCategoryItem)
	stats := make(map[string]models.CsvCategoryStat)

	for _, p := range purposes {
		counts[p] = 0
		categories[p] = make([]models.CsvVisitorCategoryItem, 0)
		stats[p] = models.CsvCategoryStat{Purpose: p}
	}

	// Query Summary Stats (Last Day & Today per Kategori dari imp_visitor)
	queryStats := `
		SELECT purpose
			, SUM(last_in) AS last_in
			, SUM(last_out) AS last_out
			, SUM(last_idle) AS last_idle
			, SUM(today_in) AS today_in
			, SUM(today_out) AS today_out
			, SUM(today_idle) AS today_idle
		FROM(
			SELECT v.purpose
				, CASE WHEN DATE(entrydate) <> CURDATE() THEN 1 ELSE 0 END AS last_in
				, CASE WHEN DATE(entrydate) <> CURDATE() AND LENGTH(exitdate) > 0 THEN 1 ELSE 0 END AS last_out
				, CASE WHEN DATE(entrydate) <> CURDATE() AND LENGTH(exitdate) = 0 THEN 1 ELSE 0 END AS last_idle
				, CASE WHEN DATE(entrydate) = CURDATE() THEN 1 ELSE 0 END AS today_in
				, CASE WHEN DATE(entrydate) = CURDATE() AND LENGTH(exitdate) > 0 THEN 1 ELSE 0 END AS today_out
				, CASE WHEN DATE(entrydate) = CURDATE() AND LENGTH(exitdate) = 0 THEN 1 ELSE 0 END AS today_idle
			FROM imp_visitor v
			WHERE (DATE(v.entrydate) = CURDATE() OR LENGTH(v.exitdate) = 0 OR DATE(v.exitdate) = CURDATE())
				AND (? = '' OR ? = 'ALL' OR ? = 'Semua Area' OR v.sitename = ?)
		) AS dt
		GROUP BY purpose
	`
	var statItems []models.CsvCategoryStat
	if err := r.db.Raw(queryStats, area, area, area, area).Scan(&statItems).Error; err == nil {
		for _, s := range statItems {
			s.TotalIdle = s.LastIdle + s.TodayIdle
			stats[s.Purpose] = s
		}
	}

	query := `
		SELECT v.sitename, CONCAT(v.nopol, ' [', v.tipe, ']') AS nopol, v.cust_code
			, DATE(v.entrydate)AS tgl_masuk, v.nopol AS no_pol
			, CASE
				WHEN v.driver = v.curr_driver THEN v.driver
				ELSE CONCAT('*', v.driver, ' -> ', v.curr_driver)
			END AS driver
			, v.category
			, DATE_FORMAT(v.entrydate, '%d-%b-%y %H:%i') AS tglmasuk
			, IFNULL(v.information, '') AS information
			, CASE 
				WHEN LENGTH(v.exitdate) > 0 AND v.purpose = 'Bongkar Muat' THEN
					CASE 
						WHEN LOWER(IFNULL(v.information,'')) LIKE '%bongkar%' OR LOWER(IFNULL(v.information,'')) LIKE '%bomgkar%' OR LOWER(IFNULL(v.information,'')) LIKE '%bonhkar%' THEN 'Selesai Bongkar'
						WHEN LOWER(IFNULL(v.information,'')) LIKE '%muat%' OR LOWER(IFNULL(v.information,'')) LIKE '%muqt%' THEN 'Selesai Muat'
						ELSE 'Selesai Bongkar Muat'
					END
				WHEN LENGTH(v.exitdate) > 0 THEN 'Selesai'
				ELSE IFNULL(wo.note, 'Antrian')
			END AS note
			, CASE 
				WHEN DATEDIFF(CURDATE(), DATE(v.entrydate)) > 0 AND LENGTH(v.exitdate) = 0 THEN CONCAT('[', DATEDIFF(CURDATE(), DATE(v.entrydate)), ' day/s late]')
				WHEN LENGTH(v.exitdate) > 0 AND v.purpose = 'Bongkar Muat' THEN
					CASE 
						WHEN LOWER(IFNULL(v.information,'')) LIKE '%bongkar%' OR LOWER(IFNULL(v.information,'')) LIKE '%bomgkar%' OR LOWER(IFNULL(v.information,'')) LIKE '%bonhkar%' THEN 'Selesai Bongkar'
						WHEN LOWER(IFNULL(v.information,'')) LIKE '%muat%' OR LOWER(IFNULL(v.information,'')) LIKE '%muqt%' THEN 'Selesai Muat'
						ELSE 'Selesai Bongkar Muat'
					END
				WHEN LENGTH(v.exitdate) > 0 THEN 'Selesai'
				ELSE IFNULL(wo.note, 'Antrian')
			END AS notelate
			, CASE WHEN LENGTH(v.exitdate) > 0 THEN '' ELSE IFNULL(wo.catt, '') END AS catt
			, DATE_FORMAT(v.exitdate, '%d-%b-%y %H:%i')AS exitdate
			, v.purpose
			, CASE WHEN DATE(v.entrydate) <> CURDATE() THEN 1 ELSE 0 END AS is_not_today
		FROM imp_visitor v
		LEFT JOIN(
			SELECT visitor_id, MAX(last_notes) AS catt, SUM(IFNULL(completed_at,0)) AS krj, tkerja
				, CASE 
					WHEN SUM(IFNULL(completed_at,0)) <> tkerja THEN 'OnProgress'
					WHEN SUM(IFNULL(completed_at,0)) = tkerja THEN 'Finished'
				END AS note
			FROM(
				SELECT visitor_id, released_at, completed_at, 1 AS tkerja
					, CASE WHEN LENGTH(IFNULL(completed_at,'')) = 0 THEN 0 ELSE 1 END AS progress
					, last_notes
				FROM imp_service_order
			) AS dt
			GROUP BY visitor_id
		) AS wo ON v.visitor_id = wo.visitor_id
		WHERE (DATE(v.entrydate) = CURDATE() OR LENGTH(v.exitdate) = 0 OR DATE(v.exitdate) = CURDATE())
			AND (? = '' OR ? = 'ALL' OR ? = 'Semua Area' OR v.sitename = ?)
			AND v.purpose IN ('Perbaikan', 'Asuransi', 'Standby', 'Bongkar Muat', 'Tunggu DO')
		ORDER BY DATE(v.exitdate), wo.note, DATE(v.entrydate) ASC
	`

	var items []models.CsvVisitorCategoryItem
	if err := r.db.Raw(query, area, area, area, area).Scan(&items).Error; err != nil {
		return nil, err
	}

	for _, item := range items {
		p := item.Purpose
		counts[p]++
		categories[p] = append(categories[p], item)
	}

	// Query Khusus Kategori Storing dari imp_service_order
	queryStoring := `
		SELECT sitename, nopol, nopol AS no_pol, cust_code, driver, category
			, DATE_FORMAT(released_at_raw, '%d-%b-%y %H:%i') AS tglmasuk
			, DATE_FORMAT(released_at_raw, '%Y-%m-%d') AS tgl_masuk
			, '' AS information
			, CASE WHEN SUM(tkerja) <> SUM(fkerja) THEN 'OnProgress' ELSE 'Finished' END AS note
			, CASE 
				WHEN DATEDIFF(CURDATE(), DATE(released_at_raw)) > 0 AND LENGTH(IFNULL(exitdate_raw,'')) = 0 THEN CONCAT('[', DATEDIFF(CURDATE(), DATE(released_at_raw)), ' day/s late]')
				WHEN LENGTH(IFNULL(exitdate_raw,'')) > 0 THEN 'Selesai'
				ELSE IFNULL(MAX(last_notes), 'Antrian')
			END AS notelate
			, CASE 
				WHEN LENGTH(IFNULL(exitdate_raw,'')) > 0 THEN '' 
				ELSE IFNULL(MAX(last_notes), '') 
			END AS catt
			, DATE_FORMAT(exitdate_raw, '%d-%b-%y %H:%i') AS exitdate
			, 'Storing' AS purpose, CASE WHEN DATE(released_at_raw) <> CURDATE() THEN 1 ELSE 0 END AS is_not_today
		FROM(
			SELECT service_order AS spk, site AS sitename, plate_number AS nopol, customer_code AS cust_code
				, '-' AS driver, category, released_at AS released_at_raw
				, 1 AS tkerja, CASE WHEN LENGTH(IFNULL(completed_at,'')) > 0 THEN 1 ELSE 0 END AS fkerja
				, last_notes, completed_at AS exitdate_raw, 'Storing' AS purpose
			FROM imp_service_order
			WHERE statue <> 'Canceled'
				AND (
					DATE(released_at) = CURDATE()
					OR LENGTH(IFNULL(DATE(completed_at),'')) < 1
				)
				AND LENGTH(IFNULL(visitor_id,'')) = 0
				AND category = 'Storing'
				AND (? = '' OR ? = 'ALL' OR ? = 'Semua Area' OR site = ?)
			ORDER BY released_at DESC
		) AS dt
		GROUP BY spk, nopol, sitename, cust_code, driver, category, released_at_raw, exitdate_raw
		ORDER BY DATE(exitdate_raw), note, released_at_raw ASC
	`
	var storingItems []models.CsvVisitorCategoryItem
	if err := r.db.Raw(queryStoring, area, area, area, area).Scan(&storingItems).Error; err == nil {
		for _, item := range storingItems {
			counts["Storing"]++
			categories["Storing"] = append(categories["Storing"], item)

			storingStat := stats["Storing"]
			if item.IsNotToday == 1 {
				storingStat.LastIn++
				if item.Exitdate != nil && *item.Exitdate != "" {
					storingStat.LastOut++
				} else {
					storingStat.LastIdle++
				}
			} else {
				storingStat.TodayIn++
				if item.Exitdate != nil && *item.Exitdate != "" {
					storingStat.TodayOut++
				} else {
					storingStat.TodayIdle++
				}
			}
			storingStat.TotalIdle = storingStat.LastIdle + storingStat.TodayIdle
			stats["Storing"] = storingStat
		}
	}

	// Query Last Update Data dari log_imp
	var lastUpdate string
	_ = r.db.Raw(`SELECT DATE_FORMAT(MAX(finished), '%d-%b-%Y %H:%i:%s') AS last_update FROM log_imp`).Scan(&lastUpdate).Error

	return &models.CsvDashboardDataResponse{
		Areas:      areas,
		Counts:     counts,
		Stats:      stats,
		Categories: categories,
		LastUpdate: lastUpdate,
	}, nil
}
