package repository

import (
	"asm/models"
	"gorm.io/gorm"
)

type SpkRepository interface {
	GetSpkPerCust(tglAwal, tglAkhir, area, cust, purpose string) ([]models.SpkPerCustItem, error)
	GetFilterOptions(tglAwal, tglAkhir, area string) (*models.SpkFilterOptions, error)
}

type spkRepository struct {
	db *gorm.DB
}

func NewSpkRepository(db *gorm.DB) SpkRepository {
	return &spkRepository{db: db}
}

func (r *spkRepository) GetSpkPerCust(tglAwal, tglAkhir, area, cust, purpose string) ([]models.SpkPerCustItem, error) {
	var items []models.SpkPerCustItem
	query := `
SELECT s.name AS site
	, so.releasedAt AS tglMasuk
	, so.completedAt AS closeSpk
	, vh.plate_number AS nopol
	, c.code AS custCode
	, so.category_value AS kategori
	, vs.purpose_value AS purpose
	, so.number AS spk
	, IFNULL(svc.service, 0) AS service
	, IFNULL(t.part, 0) AS part, IFNULL(t.ban, 0) AS ban
	, IFNULL(note.notes, '-') AS notes
	, IFNULL(pendingprd, 0) AS openskb
FROM service_orders so
LEFT JOIN visitors vs ON so.visitorId = vs.id
LEFT JOIN customers c ON so.customerId = c.id
LEFT JOIN vehicles vh ON so.vehicleId = vh.id
LEFT JOIN sites s ON so.siteId = s.id
LEFT JOIN (
	-- Subquery Biaya Jasa / Service
	SELECT so_id, SUM(tprice) AS service
	FROM (
		SELECT sod.qty * sod.price AS tprice, sod.serviceOrderId AS so_id
		FROM service_order_details sod
		WHERE sod.deletedAt IS NULL
	) AS dt
	GROUP BY so_id
) AS svc ON so.id = svc.so_id
LEFT JOIN (
	-- Subquery Biaya Sparepart & Ban
	SELECT so_id, SUM(part) AS part, SUM(ban) AS ban, SUM(pendingprd) AS pendingprd
	FROM (
		SELECT 
			sop.serviceOrderId AS so_id, 
			p.sku, 
			p.name AS prd,
			CASE WHEN ban.xbaru > 0 THEN 10000 ELSE FLOOR(sop.price) END AS unitprice,
			fi.qty,
			CASE 
				WHEN p.type = 1 THEN FLOOR(IFNULL(fi.qty, 0) * sop.price)
				ELSE FLOOR((IFNULL(ban.baru, 0) * sop.price) + (IFNULL(ban.xbaru, 0) * 10000))
			END AS tprice,
			CASE WHEN p.type = 1 THEN 'part' ELSE 'ban' END AS note,
			CASE WHEN p.type = 1 THEN FLOOR(IFNULL(fi.qty, 0) * sop.price) ELSE 0 END AS part,
			CASE WHEN p.type = 2 THEN FLOOR((IFNULL(ban.baru, 0) * sop.price) + (IFNULL(ban.xbaru, 0) * 10000)) ELSE 0 END AS ban,
			CASE WHEN sop.status_value = 'Open' THEN 1 ELSE 0 END AS pendingprd
		FROM service_order_products sop
		LEFT JOIN products p ON sop.productId = p.id
		LEFT JOIN fulfillment_items fi ON sop.id = fi.serviceOrderProductId
		LEFT JOIN (
			SELECT sopid, SUM(baru) AS baru, SUM(xbaru) AS xbaru
			FROM (
				SELECT 
					fs.serviceOrderProductId AS sopid, 
					fs.removalStatus, 
					fs.installedInventoryData,
					CASE WHEN fs.installedInventoryData->>"$.type" = 'New' THEN 1 ELSE 0 END AS baru,
					CASE WHEN fs.installedInventoryData->>"$.type" <> 'New' THEN 1 ELSE 0 END AS xbaru
				FROM fulfillment_serials fs
			) AS dt
			GROUP BY sopid
		) AS ban ON sop.id = ban.sopid
		WHERE fi.qty <> 0
	) AS dt
	GROUP BY so_id
) AS t ON so.id = t.so_id
LEFT JOIN (
	SELECT son.serviceOrderId AS so_id, son.notes
	FROM service_order_notes son
	LEFT JOIN (
		SELECT serviceOrderId, MAX(createdAt) AS mcreated
		FROM service_order_notes
		GROUP BY serviceOrderId
	) AS mx ON son.serviceOrderId = mx.serviceOrderId AND son.createdAt = mx.mcreated
	WHERE son.serviceOrderId = mx.serviceOrderId AND son.createdAt = mx.mcreated
) AS note ON so.id = note.so_id
WHERE so.` + "`status`" + ` != 10 AND so.deletedAt IS NULL AND vs.deletedAt IS NULL
	AND(
		DATE(so.releasedAt) BETWEEN ? AND ?
		OR DATE(so.completedAt) BETWEEN ? AND ?
		OR (so.completedAt IS NULL AND DATE(so.releasedAt) <= ?)
	)`

	var args []interface{}
	args = append(args, tglAwal, tglAkhir, tglAwal, tglAkhir, tglAkhir)

	if area != "" && area != "ALL" {
		query += " AND so.siteId = ?"
		args = append(args, area)
	}

	if cust != "" && cust != "ALL" {
		query += " AND so.customerId = ?"
		args = append(args, cust)
	}

	if purpose != "" && purpose != "ALL" {
		query += " AND so.category_value = ?"
		args = append(args, purpose)
	}

	query += " ORDER BY so.completedAt, so.releasedAt DESC"

	if err := r.db.Raw(query, args...).Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *spkRepository) GetFilterOptions(tglAwal, tglAkhir, area string) (*models.SpkFilterOptions, error) {
	var areas []models.AreaOption
	queryArea := `
SELECT s.id, s.name AS site
FROM sites s
LEFT JOIN (
	SELECT DISTINCT(siteId)
	FROM service_orders so
	WHERE so.` + "`status`" + ` != 10 AND so.deletedAt IS NULL
		AND(
			DATE(so.releasedAt) BETWEEN ? AND ?
			OR DATE(so.completedAt) BETWEEN ? AND ?
			OR (so.completedAt IS NULL AND DATE(so.releasedAt) <= ?)
	)
)AS so ON s.id = so.siteId
WHERE so.siteId IS NOT NULL
ORDER BY s.name
`
	if err := r.db.Raw(queryArea, tglAwal, tglAkhir, tglAwal, tglAkhir, tglAkhir).Scan(&areas).Error; err != nil {
		areas = []models.AreaOption{}
	}

	var customers []models.CustomerOption
	queryCust := `
SELECT c.id, c.code
FROM customers c
LEFT JOIN (
	SELECT DISTINCT(customerId)
	FROM service_orders so
	WHERE so.` + "`status`" + ` != 10 AND so.deletedAt IS NULL
		AND(
			DATE(so.releasedAt) BETWEEN ? AND ?
			OR DATE(so.completedAt) BETWEEN ? AND ?
			OR (so.completedAt IS NULL AND DATE(so.releasedAt) <= ?)
	)`

	var custArgs []interface{}
	custArgs = append(custArgs, tglAwal, tglAkhir, tglAwal, tglAkhir, tglAkhir)

	if area != "" && area != "ALL" {
		queryCust += ` AND siteId = ?`
		custArgs = append(custArgs, area)
	}

	queryCust += `
)AS so ON c.id = so.customerId
WHERE so.customerId IS NOT NULL
ORDER BY c.code
`
	if err := r.db.Raw(queryCust, custArgs...).Scan(&customers).Error; err != nil {
		customers = []models.CustomerOption{}
	}

	var purposes []string
	queryPurpose := `
SELECT DISTINCT(category_value) AS purpose
FROM service_orders so
WHERE
		(DATE(so.releasedAt) BETWEEN ? AND ?
		OR DATE(so.completedAt) BETWEEN ? AND ?
		OR (so.completedAt IS NULL AND DATE(so.releasedAt) <= ?))
		AND category_value IS NOT NULL AND category_value != ''
ORDER BY category_value ASC
`
	if err := r.db.Raw(queryPurpose, tglAwal, tglAkhir, tglAwal, tglAkhir, tglAkhir).Scan(&purposes).Error; err != nil {
		purposes = []string{}
	}

	return &models.SpkFilterOptions{
		Areas:     areas,
		Customers: customers,
		Purposes:  purposes,
	}, nil
}


