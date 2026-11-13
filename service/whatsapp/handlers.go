package whatsapp

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// HandleCommandSpk: /SPK [nomor_spk]
func HandleCommandSpk(db *gorm.DB, spkNumber string) string {
	spkNumber = strings.TrimSpace(spkNumber)
	if spkNumber == "" {
		return "⚠️ Format salah. Gunakan:\n*/SPK [nomor_spk]*\nContoh: `/SPK SPK-26078025`"
	}

	var data struct {
		NoSpk        string  `gorm:"column:no_spk"`
		Nopol        string  `gorm:"column:nopol"`
		Status       string  `gorm:"column:status"`
		TotalService float64 `gorm:"column:total_service"`
		TotalPart    float64 `gorm:"column:total_part"`
		Note         string  `gorm:"column:note"`
	}

	err := db.Table("spk_table").
		Select("no_spk, nopol, status, total_service, total_part, note").
		Where("no_spk = ?", spkNumber).
		First(&data).Error

	if err != nil {
		return fmt.Sprintf("❌ Data untuk SPK *%s* tidak ditemukan.", spkNumber)
	}

	status := data.Status
	if status == "" {
		status = "-"
	}
	note := strings.TrimSpace(data.Note)
	if note == "" {
		note = "-"
	}

	return fmt.Sprintf("%s [Status: %s]\nTotal Service: %s\nTotal Part: %s\nNote: %s",
		data.NoSpk, status, FormatRupiah(data.TotalService), FormatRupiah(data.TotalPart), note)
}

// HandleCommandNopol: /nopol [nomor_polisi]
func HandleCommandNopol(db *gorm.DB, nopol string) string {
	cleanNopol := strings.TrimSpace(nopol)
	if cleanNopol == "" {
		return "⚠️ Format salah. Gunakan:\n*/nopol [nomor_polisi]*\nContoh: `/nopol B 9099 JYV`"
	}

	// 1. Data Informasi Kendaraan
	sql1 := `
		SELECT v.cust_code, v.nopol, v.nobak AS cabin, v.purpose, v.tipe, v.category,
			DATE_FORMAT(v.entrydate, '%d-%b-%y %H:%i:%s') AS entrydate,
			CASE 
				WHEN v.exitdate IS NULL OR LENGTH(TRIM(v.exitdate)) = 0 THEN 'OnProgress' 
				ELSE DATE_FORMAT(v.exitdate, '%d-%b-%y %H:%i:%s') 
			END AS exitdate,
			v.information
		FROM imp_visitor v
		WHERE REPLACE(v.nopol, ' ', '') = REPLACE(?, ' ', '')
		ORDER BY id DESC
		LIMIT 1
	`
	var q1 struct {
		CustCode    string `gorm:"column:cust_code"`
		Nopol       string `gorm:"column:nopol"`
		Cabin       string `gorm:"column:cabin"`
		Purpose     string `gorm:"column:purpose"`
		Tipe        string `gorm:"column:tipe"`
		Category    string `gorm:"column:category"`
		EntryDate   string `gorm:"column:entrydate"`
		ExitDate    string `gorm:"column:exitdate"`
		Information string `gorm:"column:information"`
	}

	if err := db.Raw(sql1, cleanNopol).Scan(&q1).Error; err != nil || q1.Nopol == "" {
		return fmt.Sprintf("❌ Data kendaraan untuk nopol *%s* tidak ditemukan.", nopol)
	}

	// 2. Data SPK dan Biaya
	sql2 := `
		SELECT service_order, MAX(category) AS category, MIN(DATE_FORMAT(released_at, '%d-%b-%y %H:%i:%s')) AS mulai, IFNULL(MAX(completed_at), 'OnProgress') AS selesai,
			SUM(service) AS service, SUM(part) AS part, SUM(ban) AS ban
		FROM(
			SELECT so.category, so.service_order, so.reff, so.released_at, so.completed_at,
				CASE WHEN tipe = 'Services' THEN so.total_price ELSE 0 END AS service,
				CASE WHEN tipe = 'Parts' THEN so.total_price ELSE 0 END AS part,
				CASE WHEN tipe = 'Inventories' THEN so.total_price ELSE 0 END AS ban,
				so.visitor_id, v.id
			FROM imp_service_order so
			INNER JOIN(
				SELECT visitor_id AS id
				FROM imp_visitor v
				WHERE REPLACE(v.nopol, ' ', '') = REPLACE(?, ' ', '')
					AND (
						DATE(v.entrydate) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
						OR (DATE(v.exitdate) IS NULL AND DATE(v.entrydate) <= LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY)
						OR DATE(v.exitdate) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
					)
				ORDER BY v.entrydate DESC
			) AS v ON so.visitor_id = v.id
		) AS dt
		GROUP BY service_order
	`
	type spkRow struct {
		ServiceOrder string  `gorm:"column:service_order"`
		Category     string  `gorm:"column:category"`
		Mulai        string  `gorm:"column:mulai"`
		Selesai      string  `gorm:"column:selesai"`
		Service      float64 `gorm:"column:service"`
		Part         float64 `gorm:"column:part"`
		Ban          float64 `gorm:"column:ban"`
	}
	var q2 []spkRow
	db.Raw(sql2, cleanNopol).Scan(&q2)

	var totalService, totalPart, totalBan float64
	var spkList []string

	for _, r := range q2 {
		isDone := strings.TrimSpace(r.Selesai) != "" && len(strings.TrimSpace(r.Selesai)) > 12 && r.Selesai != "OnProgress"
		doneTag := ""
		if isDone {
			doneTag = " [DONE]"
		}
		spkList = append(spkList, "- "+r.ServiceOrder+doneTag)
		totalService += r.Service
		totalPart += r.Part
		totalBan += r.Ban
	}

	grandTotal := totalService + totalPart + totalBan
	spkText := "- Tidak ada SPK"
	if len(spkList) > 0 {
		spkText = strings.Join(spkList, "\n")
	}

	cabinText := ""
	if q1.Cabin != "" {
		cabinText = fmt.Sprintf("[%s]", q1.Cabin)
	}
	infoText := strings.TrimSpace(q1.Information)
	if infoText == "" {
		infoText = "-"
	}

	return fmt.Sprintf("*Informasi Kendaraan*\nPurpose: %s - %s\nNopol: *%s %s*\nTipe: %s - %s\nKunjungan terakhir: %s\nExit: %s\nInformasi: %s\n------------------------------\nTotal SPK : %d\n%s\n\n👨‍🔧 %s\n🛠️ %s\n🛞 %s\nTotal: %s",
		q1.CustCode, q1.Purpose, q1.Nopol, cabinText, q1.Tipe, q1.Category, q1.EntryDate, q1.ExitDate, infoText, len(q2), spkText,
		FormatRupiah(totalService), FormatRupiah(totalPart), FormatRupiah(totalBan), FormatRupiah(grandTotal))
}

// HandleCommandOmsetClosing: /omsetclosing
func HandleCommandOmsetClosing(db *gorm.DB) string {
	where := " WHERE DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) "
	day := time.Now().Day()
	if day >= 1 && day <= 10 {
		where = " WHERE DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 2 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE() - INTERVAL 1 MONTH) "
	}

	sql := fmt.Sprintf(`
		SELECT cust, COUNT(so) AS spk,
			FLOOR(SUM(service)) AS service,
			FLOOR(SUM(part)) AS part,
			FLOOR(SUM(ban)) AS ban,
			FLOOR(SUM(total)) AS total
		FROM(
			SELECT cust, so,
				FLOOR(SUM(service)) AS service,
				FLOOR(SUM(part)) AS part,
				FLOOR(SUM(ban)) AS ban,
				FLOOR(SUM(total)) AS total
			FROM(
				SELECT service_order AS so, customer_code AS cust,
					CASE WHEN tipe = 'Services' THEN total_price ELSE 0 END AS service,
					CASE WHEN tipe = 'Parts' THEN total_price ELSE 0 END AS part,
					CASE WHEN tipe = 'Inventories' THEN total_price ELSE 0 END AS ban,
					total_price AS total,
					1 AS spk
				FROM imp_service_order 
				%s
			) AS dt
			GROUP BY cust, so
		) AS dt
		GROUP BY cust
		ORDER BY total DESC
	`, where)

	type row struct {
		Cust    string  `gorm:"column:cust"`
		Spk     int     `gorm:"column:spk"`
		Service float64 `gorm:"column:service"`
		Part    float64 `gorm:"column:part"`
		Ban     float64 `gorm:"column:ban"`
		Total   float64 `gorm:"column:total"`
	}
	var rows []row
	if err := db.Raw(sql).Scan(&rows).Error; err != nil || len(rows) == 0 {
		return "❌ Tidak ada data omset closing untuk periode bulan ini."
	}

	var itemsText []string
	var totalService, totalPart, totalBan, grandTotal float64

	for _, r := range rows {
		totalService += r.Service
		totalPart += r.Part
		totalBan += r.Ban
		grandTotal += r.Total

		item := fmt.Sprintf("*%s* (%d SPK)\n👨‍🔧 %s\n🛠️ %s\n🛞 %s\n*∑ %s*",
			r.Cust, r.Spk, FormatRupiah(r.Service), FormatRupiah(r.Part), FormatRupiah(r.Ban), FormatRupiah(r.Total))
		itemsText = append(itemsText, item)
	}

	return fmt.Sprintf("%s\n\n------------------------------\n*TOTAL:*\n👨‍🔧 %s\n🛠️ %s\n🛞 %s\n∑ *%s*",
		strings.Join(itemsText, "\n\n"),
		FormatRupiah(totalService), FormatRupiah(totalPart), FormatRupiah(totalBan), FormatRupiah(grandTotal))
}

// HandleCommandOmset: /omset [customer_code / ALL]
func HandleCommandOmset(db *gorm.DB, parameter string) string {
	param := strings.ToUpper(strings.TrimSpace(parameter))
	if param == "" || param == "ALL" {
		return HandleCommandOmsetAll(db)
	}
	return HandleCommandOmsetCust(db, param)
}

// HandleCommandOmsetAll: /omset ALL
func HandleCommandOmsetAll(db *gorm.DB) string {
	sql := `
		SELECT cust, COUNT(so) AS spk, SUM(soc) AS close_spk,
			COUNT(so) - SUM(soc) AS pending_spk,
			FLOOR(SUM(total_service)) AS total_service,
			FLOOR(SUM(total_part)) AS total_part,
			FLOOR(SUM(total_ban)) AS total_ban,
			FLOOR(SUM(total_total)) AS total_total,
			FLOOR(SUM(close_service)) AS close_service,
			FLOOR(SUM(close_part)) AS close_part,
			FLOOR(SUM(close_ban)) AS close_ban,
			FLOOR(SUM(close_total)) AS close_total,
			FLOOR(SUM(total_service)) - FLOOR(SUM(close_service)) AS pending_service,
			FLOOR(SUM(total_part)) - FLOOR(SUM(close_part)) AS pending_part,
			FLOOR(SUM(total_ban)) - FLOOR(SUM(close_ban)) AS pending_ban,
			FLOOR(SUM(total_total)) - FLOOR(SUM(close_total)) AS pending_total
		FROM(
			SELECT cust, so, CASE WHEN LENGTH(so_close) > 1 THEN 1 ELSE 0 END AS soc,
				FLOOR(SUM(tservice)) AS total_service,
				FLOOR(SUM(tpart)) AS total_part,
				FLOOR(SUM(tban)) AS total_ban,
				FLOOR(SUM(ttotal)) AS total_total,
				FLOOR(SUM(cservice)) AS close_service,
				FLOOR(SUM(cpart)) AS close_part,
				FLOOR(SUM(cban)) AS close_ban,
				FLOOR(SUM(ctotal)) AS close_total
			FROM(
				SELECT service_order AS so, customer_code AS cust,
					CASE WHEN tipe = 'Services' THEN total_price ELSE 0 END AS tservice,
					CASE WHEN tipe = 'Parts' THEN total_price ELSE 0 END AS tpart,
					CASE WHEN tipe = 'Inventories' THEN total_price ELSE 0 END AS tban,
					total_price AS ttotal,
					CASE WHEN DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) AND tipe = 'Services' THEN total_price ELSE 0 END AS cservice,
					CASE WHEN DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) AND tipe = 'Parts' THEN total_price ELSE 0 END AS cpart,
					CASE WHEN DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) AND tipe = 'Inventories' THEN total_price ELSE 0 END AS cban,
					CASE WHEN DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) THEN total_price ELSE 0 END AS ctotal,
					CASE WHEN DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) THEN service_order ELSE 0 END AS so_close
				FROM imp_service_order so
				WHERE (
					DATE(so.released_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
					OR (DATE(so.completed_at) IS NULL AND DATE(so.released_at) < LAST_DAY(CURDATE() - INTERVAL 1 MONTH))
					OR DATE(so.completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
					)
				ORDER BY so.id DESC
			) AS dt
			GROUP BY cust, so, so_close
		) AS dt
		GROUP BY cust
		ORDER BY close_total DESC
	`

	type row struct {
		Cust           string  `gorm:"column:cust"`
		Spk            int     `gorm:"column:spk"`
		CloseSpk       int     `gorm:"column:close_spk"`
		PendingSpk     int     `gorm:"column:pending_spk"`
		CloseService   float64 `gorm:"column:close_service"`
		ClosePart      float64 `gorm:"column:close_part"`
		CloseBan       float64 `gorm:"column:close_ban"`
		CloseTotal     float64 `gorm:"column:close_total"`
		PendingService float64 `gorm:"column:pending_service"`
		PendingPart    float64 `gorm:"column:pending_part"`
		PendingBan     float64 `gorm:"column:pending_ban"`
		PendingTotal   float64 `gorm:"column:pending_total"`
	}
	var rows []row
	if err := db.Raw(sql).Scan(&rows).Error; err != nil || len(rows) == 0 {
		return "❌ Tidak ada data omset untuk periode bulan ini."
	}

	var gtCloseService, gtPendingService, gtClosePart, gtPendingPart, gtCloseBan, gtPendingBan, gtCloseTotal, gtPendingTotal float64
	var blocks []string

	for idx, r := range rows {
		gtCloseService += r.CloseService
		gtPendingService += r.PendingService
		gtClosePart += r.ClosePart
		gtPendingPart += r.PendingPart
		gtCloseBan += r.CloseBan
		gtPendingBan += r.PendingBan
		gtCloseTotal += r.CloseTotal
		gtPendingTotal += r.PendingTotal

		b := fmt.Sprintf("🏢 *%d. %s*\n(%d SPK|%d SPK)\n------------------------------\n👨‍🔧 %s|%s\n🛠️ %s|%s\n🛞 %s|%s\n📋 %s|%s",
			idx+1, r.Cust, r.CloseSpk, r.PendingSpk,
			FormatRupiah(r.CloseService), FormatRupiah(r.PendingService),
			FormatRupiah(r.ClosePart), FormatRupiah(r.PendingPart),
			FormatRupiah(r.CloseBan), FormatRupiah(r.PendingBan),
			FormatRupiah(r.CloseTotal), FormatRupiah(r.PendingTotal))
		blocks = append(blocks, b)
	}

	header := fmt.Sprintf("📊 *REKAP OMSET %d CUSTOMER*\n_(Format: Selesai | Pending)_", len(rows))
	grandTotal := fmt.Sprintf("📋 *GRAND TOTAL (%d CUST)*\n👨‍🔧 %s | %s\n🛠️ %s | %s\n🛞 %s | %s\n📋 %s | %s",
		len(rows),
		FormatRupiah(gtCloseService), FormatRupiah(gtPendingService),
		FormatRupiah(gtClosePart), FormatRupiah(gtPendingPart),
		FormatRupiah(gtCloseBan), FormatRupiah(gtPendingBan),
		FormatRupiah(gtCloseTotal), FormatRupiah(gtPendingTotal))

	return header + "\n==============================\n" +
		strings.Join(blocks, "\n==============================\n") +
		"\n==============================\n" + grandTotal
}

// HandleCommandOmsetCust: /omset [custCode]
func HandleCommandOmsetCust(db *gorm.DB, custCode string) string {
	sql := `
		SELECT cust, COUNT(so) AS spk, SUM(soc) AS close_spk,
			COUNT(so) - SUM(soc) AS pending_spk,
			FLOOR(SUM(total_service)) AS total_service,
			FLOOR(SUM(total_part)) AS total_part,
			FLOOR(SUM(total_ban)) AS total_ban,
			FLOOR(SUM(total_total)) AS total_total,
			FLOOR(SUM(close_service)) AS close_service,
			FLOOR(SUM(close_part)) AS close_part,
			FLOOR(SUM(close_ban)) AS close_ban,
			FLOOR(SUM(close_total)) AS close_total,
			FLOOR(SUM(total_service)) - FLOOR(SUM(close_service)) AS pending_service,
			FLOOR(SUM(total_part)) - FLOOR(SUM(close_part)) AS pending_part,
			FLOOR(SUM(total_ban)) - FLOOR(SUM(close_ban)) AS pending_ban,
			FLOOR(SUM(total_total)) - FLOOR(SUM(close_total)) AS pending_total
		FROM(
			SELECT cust, so, CASE WHEN LENGTH(so_close) > 1 THEN 1 ELSE 0 END AS soc,
				FLOOR(SUM(tservice)) AS total_service,
				FLOOR(SUM(tpart)) AS total_part,
				FLOOR(SUM(tban)) AS total_ban,
				FLOOR(SUM(ttotal)) AS total_total,
				FLOOR(SUM(cservice)) AS close_service,
				FLOOR(SUM(cpart)) AS close_part,
				FLOOR(SUM(cban)) AS close_ban,
				FLOOR(SUM(ctotal)) AS close_total
			FROM(
				SELECT service_order AS so, customer_code AS cust,
					CASE WHEN tipe = 'Services' THEN total_price ELSE 0 END AS tservice,
					CASE WHEN tipe = 'Parts' THEN total_price ELSE 0 END AS tpart,
					CASE WHEN tipe = 'Inventories' THEN total_price ELSE 0 END AS tban,
					total_price AS ttotal,
					CASE WHEN DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) AND tipe = 'Services' THEN total_price ELSE 0 END AS cservice,
					CASE WHEN DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) AND tipe = 'Parts' THEN total_price ELSE 0 END AS cpart,
					CASE WHEN DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) AND tipe = 'Inventories' THEN total_price ELSE 0 END AS cban,
					CASE WHEN DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) THEN total_price ELSE 0 END AS ctotal,
					CASE WHEN DATE(completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE()) THEN service_order ELSE 0 END AS so_close
				FROM imp_service_order so
				WHERE (
					DATE(so.released_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
					OR (DATE(so.completed_at) IS NULL AND DATE(so.released_at) < LAST_DAY(CURDATE() - INTERVAL 1 MONTH))
					OR DATE(so.completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
					) AND customer_code = ?
				ORDER BY so.id DESC
			) AS dt
			GROUP BY cust, so, so_close
		) AS dt
		GROUP BY cust
	`

	var r struct {
		Cust           string  `gorm:"column:cust"`
		Spk            int     `gorm:"column:spk"`
		CloseSpk       int     `gorm:"column:close_spk"`
		PendingSpk     int     `gorm:"column:pending_spk"`
		CloseService   float64 `gorm:"column:close_service"`
		ClosePart      float64 `gorm:"column:close_part"`
		CloseBan       float64 `gorm:"column:close_ban"`
		CloseTotal     float64 `gorm:"column:close_total"`
		PendingService float64 `gorm:"column:pending_service"`
		PendingPart    float64 `gorm:"column:pending_part"`
		PendingBan     float64 `gorm:"column:pending_ban"`
		PendingTotal   float64 `gorm:"column:pending_total"`
	}

	if err := db.Raw(sql, custCode).Scan(&r).Error; err != nil || r.Cust == "" {
		return fmt.Sprintf("❌ Data omset untuk customer *%s* tidak ditemukan.", custCode)
	}

	return fmt.Sprintf("🏢 *LAPORAN OMSET %s*\n(%d SPK|%d SPK)\n------------------------------\n👨‍🔧 %s|%s\n🛠️ %s|%s\n🛞 %s|%s\n📋 %s|%s",
		r.Cust, r.CloseSpk, r.PendingSpk,
		FormatRupiah(r.CloseService), FormatRupiah(r.PendingService),
		FormatRupiah(r.ClosePart), FormatRupiah(r.PendingPart),
		FormatRupiah(r.CloseBan), FormatRupiah(r.PendingBan),
		FormatRupiah(r.CloseTotal), FormatRupiah(r.PendingTotal))
}

// HandleCommandOmsetPending: /omsetpending [custCode]
func HandleCommandOmsetPending(db *gorm.DB, custCode string) string {
	custCode = strings.ToUpper(strings.TrimSpace(custCode))
	if custCode != "" {
		return HandleCommandOmsetPendingCust(db, custCode)
	}

	sql := `
		SELECT cust, COUNT(so) AS spk,
			FLOOR(SUM(service)) AS service,
			FLOOR(SUM(part)) AS part,
			FLOOR(SUM(ban)) AS ban,
			FLOOR(SUM(total)) AS total
		FROM(
			SELECT cust, so,
				FLOOR(SUM(service)) AS service,
				FLOOR(SUM(part)) AS part,
				FLOOR(SUM(ban)) AS ban,
				FLOOR(SUM(total)) AS total
			FROM(
				SELECT service_order AS so, customer_code AS cust,
					CASE WHEN tipe = 'Services' THEN total_price ELSE 0 END AS service,
					CASE WHEN tipe = 'Parts' THEN total_price ELSE 0 END AS part,
					CASE WHEN tipe = 'Inventories' THEN total_price ELSE 0 END AS ban,
					total_price AS total
				FROM imp_service_order so
				WHERE (
					DATE(so.released_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
					OR (DATE(so.completed_at) IS NULL AND DATE(so.released_at) < LAST_DAY(CURDATE() - INTERVAL 1 MONTH))
					OR DATE(so.completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
					)
					AND DATE(completed_at) IS NULL 
			) AS dt
			GROUP BY cust, so
		) AS dt
		GROUP BY cust
		ORDER BY total DESC
	`

	type row struct {
		Cust    string  `gorm:"column:cust"`
		Spk     int     `gorm:"column:spk"`
		Service float64 `gorm:"column:service"`
		Part    float64 `gorm:"column:part"`
		Ban     float64 `gorm:"column:ban"`
		Total   float64 `gorm:"column:total"`
	}
	var rows []row
	if err := db.Raw(sql).Scan(&rows).Error; err != nil || len(rows) == 0 {
		return "❌ Tidak ada data omset pending untuk periode bulan ini."
	}

	var itemsText []string
	var totalService, totalPart, totalBan, grandTotal float64

	for _, r := range rows {
		totalService += r.Service
		totalPart += r.Part
		totalBan += r.Ban
		grandTotal += r.Total

		item := fmt.Sprintf("*%s* (%d SPK)\n👨‍🔧 %s\n🛠️ %s\n🛞 %s\n*∑ %s*",
			r.Cust, r.Spk, FormatRupiah(r.Service), FormatRupiah(r.Part), FormatRupiah(r.Ban), FormatRupiah(r.Total))
		itemsText = append(itemsText, item)
	}

	return fmt.Sprintf("OMSET PENDING\n------------------------------\n%s\n==============================\n*TOTAL:*\n👨‍🔧 %s\n🛠️ %s\n🛞 %s\n∑ *%s*",
		strings.Join(itemsText, "\n------------------------------\n"),
		FormatRupiah(totalService), FormatRupiah(totalPart), FormatRupiah(totalBan), FormatRupiah(grandTotal))
}

// HandleCommandOmsetPendingCust: /omsetpending [custCode] detail
func HandleCommandOmsetPendingCust(db *gorm.DB, custCode string) string {
	sql := `
		SELECT nopol, COUNT(so) AS spk, MAX(cat) AS kategori,
			FLOOR(SUM(service)) AS service,
			FLOOR(SUM(part)) AS part,
			FLOOR(SUM(ban)) AS ban,
			FLOOR(SUM(total)) AS total,
			MAX(note) AS note
		FROM(
			SELECT nopol, so, MAX(category) AS cat,
				FLOOR(SUM(service)) AS service,
				FLOOR(SUM(part)) AS part,
				FLOOR(SUM(ban)) AS ban,
				FLOOR(SUM(total)) AS total,
				MAX(note) AS note
			FROM(
				SELECT service_order AS so, customer_code AS cust, plate_number AS nopol,
					CASE WHEN tipe = 'Services' THEN total_price ELSE 0 END AS service,
					CASE WHEN tipe = 'Parts' THEN total_price ELSE 0 END AS part,
					CASE WHEN tipe = 'Inventories' THEN total_price ELSE 0 END AS ban,
					total_price AS total, category,
					last_notes AS note
				FROM imp_service_order so
				WHERE (
					DATE(so.released_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
					OR (DATE(so.completed_at) IS NULL AND DATE(so.released_at) < LAST_DAY(CURDATE() - INTERVAL 1 MONTH))
					OR DATE(so.completed_at) BETWEEN LAST_DAY(CURDATE() - INTERVAL 1 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
					)
					AND DATE(completed_at) IS NULL
					AND customer_code = ?
			) AS dt
			GROUP BY nopol, so
			ORDER BY total DESC
		) AS dt
		GROUP BY nopol
		ORDER BY total DESC
	`

	type row struct {
		Nopol    string  `gorm:"column:nopol"`
		Spk      int     `gorm:"column:spk"`
		Kategori string  `gorm:"column:kategori"`
		Service  float64 `gorm:"column:service"`
		Part     float64 `gorm:"column:part"`
		Ban      float64 `gorm:"column:ban"`
		Total    float64 `gorm:"column:total"`
		Note     string  `gorm:"column:note"`
	}
	var rows []row
	if err := db.Raw(sql, custCode).Scan(&rows).Error; err != nil || len(rows) == 0 {
		return fmt.Sprintf("❌ Tidak ada data omset pending untuk customer *%s*.", custCode)
	}

	var itemsText []string
	var totalService, totalPart, totalBan, grandTotal float64

	for _, r := range rows {
		totalService += r.Service
		totalPart += r.Part
		totalBan += r.Ban
		grandTotal += r.Total

		var lines []string
		lines = append(lines, fmt.Sprintf("*%s* (%d SPK) [%s]", r.Nopol, r.Spk, r.Kategori))
		if r.Service > 0 {
			lines = append(lines, "👨‍🔧 "+FormatRupiah(r.Service))
		}
		if r.Part > 0 {
			lines = append(lines, "🛠️ "+FormatRupiah(r.Part))
		}
		if r.Ban > 0 {
			lines = append(lines, "🛞 "+FormatRupiah(r.Ban))
		}
		lines = append(lines, "∑ *"+FormatRupiah(r.Total)+"*")
		if r.Note != "" && r.Note != "-" {
			lines = append(lines, "*)Note: "+r.Note)
		}

		itemsText = append(itemsText, strings.Join(lines, "\n"))
	}

	return fmt.Sprintf("*TOTAL PENDING %s : %d DATA*\n------------------------------\n%s\n------------------------------\n*TOTAL:*\n👨‍🔧 %s\n🛠️ %s\n🛞 %s\n∑ *%s*",
		custCode, len(rows),
		strings.Join(itemsText, "\n------------------------------\n"),
		FormatRupiah(totalService), FormatRupiah(totalPart), FormatRupiah(totalBan), FormatRupiah(grandTotal))
}

// HandleCommandInfoLelang: /infolelang [nopol]
func HandleCommandInfoLelang(db *gorm.DB, parameter string) string {
	param := strings.TrimSpace(parameter)

	// Jika ada parameter nopol
	if param != "" {
		cleanParam := strings.ReplaceAll(strings.ToUpper(param), " ", "")
		var row struct {
			Nopol string `gorm:"column:nopol"`
			Note  string `gorm:"column:note"`
		}
		err := db.Table("prior_unit").
			Where("kategori = 'lelang' AND REPLACE(nopol, ' ', '') = ?", cleanParam).
			Order("id DESC").
			First(&row).Error

		if err != nil {
			return "Belum ada lelang dengan nopol tersebut"
		}
		note := strings.TrimSpace(row.Note)
		if note == "" {
			note = "-"
		}
		return fmt.Sprintf("Update Lelang %s\n%s", row.Nopol, note)
	}

	// Rekap keseluruhan
	type unitRow struct {
		ID     int64  `gorm:"column:id"`
		Nopol  string `gorm:"column:nopol"`
		Cabin  string `gorm:"column:cabin"`
		Jenis  string `gorm:"column:jenis"`
		Statue string `gorm:"column:statue"`
		Note   string `gorm:"column:note"`
	}
	var rows []unitRow
	err := db.Table("prior_unit").
		Where("kategori = 'lelang'").
		Order("id ASC").
		Find(&rows).Error

	if err != nil || len(rows) == 0 {
		return "❌ Tidak ada data unit lelang saat ini."
	}

	type group struct {
		selesai      []unitRow
		belumSelesai []unitRow
	}
	grouped := make(map[string]*group)
	var jenisOrder []string

	for _, r := range rows {
		jenis := strings.ToUpper(strings.TrimSpace(r.Jenis))
		if jenis == "" {
			jenis = "LAINNYA"
		}
		if _, exists := grouped[jenis]; !exists {
			grouped[jenis] = &group{}
			jenisOrder = append(jenisOrder, jenis)
		}
		if r.Statue == "1" {
			grouped[jenis].selesai = append(grouped[jenis].selesai, r)
		} else {
			grouped[jenis].belumSelesai = append(grouped[jenis].belumSelesai, r)
		}
	}

	var sections []string
	for _, j := range jenisOrder {
		g := grouped[j]
		totalUnit := len(g.selesai) + len(g.belumSelesai)

		var lines []string
		lines = append(lines, fmt.Sprintf("*PROGRES UNIT LELANG %s (%d UNIT)*\n", j, totalUnit))

		// Selesai
		lines = append(lines, fmt.Sprintf("* SELESAI = %d UNIT", len(g.selesai)))
		if len(g.selesai) > 0 {
			for idx, u := range g.selesai {
				lines = append(lines, fmt.Sprintf("%d. %s / %s", idx+1, strings.TrimSpace(u.Nopol), strings.TrimSpace(u.Cabin)))
			}
		} else {
			lines = append(lines, "-")
		}

		lines = append(lines, "")

		// Belum Selesai
		labelBelum := "BELUM SELESAI"
		if j == "FIGHTER" {
			labelBelum = "DALAM PENGERJAAN"
		}
		lines = append(lines, fmt.Sprintf("* %s = %d UNIT", labelBelum, len(g.belumSelesai)))
		if len(g.belumSelesai) > 0 {
			for idx, u := range g.belumSelesai {
				note := strings.TrimSpace(u.Note)
				noteText := ""
				if note != "" && strings.ToUpper(note) != "DONE" {
					noteText = " " + note
				}
				lines = append(lines, fmt.Sprintf("%d. %s / %s%s", idx+1, strings.TrimSpace(u.Nopol), strings.TrimSpace(u.Cabin), noteText))
			}
		} else {
			lines = append(lines, "-")
		}

		sections = append(sections, strings.Join(lines, "\n"))
	}

	return strings.Join(sections, "\n\n")
}

// HandleCommandUpdateLelang: /updatelelang [nopol]\n[note...]
func HandleCommandUpdateLelang(db *gorm.DB, parameter string) string {
	text := strings.TrimSpace(parameter)
	if text == "" {
		return "⚠️ Format salah. Gunakan:\n*/updatelelang [nopol]*\n[Catatan update...]"
	}

	lines := strings.Split(text, "\n")
	firstLine := strings.TrimSpace(lines[0])
	var noteBody string
	if len(lines) > 1 {
		noteBody = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}

	var nopolTarget string
	if noteBody == "" {
		re := regexp.MustCompile(`(?i)^([A-Z]{1,2}\s*\d{1,4}\s*[A-Z]{1,3})\s+(.+)$`)
		matches := re.FindStringSubmatch(firstLine)
		if len(matches) == 3 {
			nopolTarget = strings.TrimSpace(matches[1])
			noteBody = strings.TrimSpace(matches[2])
		} else {
			nopolTarget = firstLine
		}
	} else {
		nopolTarget = firstLine
	}

	if nopolTarget == "" || noteBody == "" {
		return "⚠️ Mohon sertakan nomor polisi dan catatan update lelang."
	}

	cleanNopol := strings.ReplaceAll(strings.ToUpper(nopolTarget), " ", "")

	var row struct {
		ID     int64  `gorm:"column:id"`
		Nopol  string `gorm:"column:nopol"`
		Statue string `gorm:"column:statue"`
	}

	err := db.Table("prior_unit").
		Where("kategori = 'lelang' AND REPLACE(nopol, ' ', '') = ?", cleanNopol).
		Order("id DESC").
		First(&row).Error

	if err != nil {
		return "Belum ada lelang dengan nopol tersebut"
	}

	if row.Statue == "1" {
		return "Nopol Tersebut sudah DONE"
	}

	db.Table("prior_unit").
		Where("id = ? AND statue = 0", row.ID).
		Update("note", noteBody)

	return fmt.Sprintf("✅ Catatan lelang *%s* berhasil diperbarui:\n%s", row.Nopol, noteBody)
}

// HandleCommandHelp: /help atau /bantuan
func HandleCommandHelp() string {
	return `🤖 *MENU BANTUAN BOT WHATSAPP*

Berikut perintah yang tersedia:
1. */SPK [nomor_spk]*
   Mengecek rincian biaya SPK.
   Contoh: ` + "`/SPK SPK-26078025`" + `

2. */nopol [nomor_polisi]*
   Mengecek SPK terakhir kendaraan.
   Contoh: ` + "`/nopol B 9378 JYV`" + `

3. */omsetclosing*
   Mengecek rekapitulasi omset closing bulan ini per customer.

4. */omset [customer_code]*
   Mengecek rekapitulasi omset closing vs pending secara keseluruhan atau per customer.
   Contoh: ` + "`/omset` atau `/omset NCA`" + `

5. */omsetpending [customer_code]*
   Mengecek rekapitulasi omset pending secara keseluruhan atau per customer.
   Contoh: ` + "`/omsetpending` atau `/omsetpending BAM`" + `

6. */infolelang [nopol]*
   Mengecek progres unit prioritas lelang (keseluruhan atau per nopol).
   Contoh: ` + "`/infolelang` atau `/infolelang B 9625 JYU`" + `

7. */updatelelang [nopol]*
   Memperbarui catatan/note progres unit lelang.

8. */registrasi*
   Mendaftar untuk aktivasi penggunaan bot WA AutoResponse.

9. */help*
   Menampilkan daftar menu bantuan ini.`
}

// HandleCommandHelpAdmin: /helpadmin
func HandleCommandHelpAdmin(db *gorm.DB, senderPhone string) string {
	if !IsAdmin(db, senderPhone) {
		return "⛔ Maaf, perintah */helpadmin* hanya dapat diakses oleh Admin."
	}

	return `👑 *MENU BANTUAN KHUSUS ADMIN*

Berikut perintah operasional khusus Admin:
1. */aktifasi [no_hp] [token_pass]*
   Mengaktifkan akun pengguna baru yang sudah mendaftar via ` + "`/registrasi`" + `.
   Contoh: ` + "`/aktifasi 081808568123 112233`" + `

2. */tarikdata* atau */sync*
   Memaksa proses penarikan dan pembaruan data spreadsheet ke database secara instan.
   Contoh: ` + "`/tarikdata`" + `

3. */help*
   Melihat daftar perintah umum bot.

4. */helpadmin*
   Menampilkan menu bantuan khusus admin ini.`
}

// HandleCommandRegistrasi: /registrasi
func HandleCommandRegistrasi(db *gorm.DB, bot *WhatsAppBot, senderPhone string) string {
	_, p0, p62 := CleanPhone(senderPhone)

	var user struct {
		ID     int64 `gorm:"column:id"`
		Statue int   `gorm:"column:statue"`
	}

	err := db.Table("users").
		Select("id, statue").
		Where("no_hp = ? OR no_hp = ?", p0, p62).
		First(&user).Error

	if err == nil {
		if user.Statue == 9 {
			return "Anda sudah terdaftar, silahkan ketik /help untuk menggunakan fitur ini."
		}
		if user.Statue == 0 {
			return "Anda sudah masuk dalam list antrian kami, silahkan follow up ke admin kami."
		}
		if user.Statue == 4 {
			return "Status akun Anda sudah inactive, mohon hubungi admin kami untuk konfirmasi lebih lanjut."
		}
		return fmt.Sprintf("Status akun Anda: %d. Silahkan hubungi admin kami.", user.Statue)
	}

	// Daftarkan ke tabel users
	newUser := map[string]interface{}{
		"no_hp":    p0,
		"statue":   0,
		"approved": "0",
	}
	db.Table("users").Create(&newUser)

	// Notifikasi admin di tabel settings
	var admins []struct {
		NoHp string `gorm:"column:no_hp"`
	}
	db.Table("settings").
		Select("no_hp").
		Where("no_hp IS NOT NULL AND no_hp != ''").
		Find(&admins)

	adminNotifyMsg := fmt.Sprintf("%s request akses WA AutoResponder", p0)
	for _, a := range admins {
		go bot.SendMessage(a.NoHp, adminNotifyMsg)
	}

	return "Anda mengaktifkan fitur WA Auto Response kami, mohon menunggu sampai aktifasi akun Anda diaktifkan."
}

// HandleCommandAktifasi: /aktifasi [no_hp] [getpass]
func HandleCommandAktifasi(db *gorm.DB, bot *WhatsAppBot, senderPhone, parameter string) string {
	params := strings.Fields(strings.TrimSpace(parameter))
	if len(params) < 2 {
		return "⚠️ Format salah. Gunakan:\n*/aktifasi [no_hp] [getpass]*\nContoh: `/aktifasi 081808568123 112233`"
	}

	targetPhone := params[0]
	token := params[1]

	_, s0, s62 := CleanPhone(senderPhone)

	var admin struct {
		NoHp    string         `gorm:"column:no_hp"`
		Getpass sql.NullString `gorm:"column:getpass"`
	}
	err := db.Table("settings").
		Where("no_hp = ? OR no_hp = ?", s0, s62).
		First(&admin).Error

	if err != nil {
		return "Anda bukan admin approval aktifasi user."
	}

	_, t0, t62 := CleanPhone(targetPhone)
	var targetUser struct {
		ID int64 `gorm:"column:id"`
	}
	err = db.Table("users").
		Where("no_hp = ? OR no_hp = ?", t0, t62).
		First(&targetUser).Error

	if err != nil {
		return "Nomor WA tersebut belum terdaftar pada system ini"
	}

	if !admin.Getpass.Valid || admin.Getpass.String != token {
		return "Wrong token pass.."
	}

	db.Table("users").
		Where("id = ?", targetUser.ID).
		Updates(map[string]interface{}{
			"statue":   9,
			"approved": admin.NoHp,
		})

	// Notifikasi ke nomor user yang baru diaktifkan
	go bot.SendMessage(targetPhone, "Nomor HP Anda sudah terhubung dengan WA AutoResponse kami, silahkan ketik /help untuk menggunakan fitur ini.")

	return fmt.Sprintf("✅ Nomor *%s* berhasil diaktifkan dengan status 9.", t0)
}
