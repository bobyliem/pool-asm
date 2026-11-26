package repository

import (
	"strings"

	"asm/models"

	"gorm.io/gorm"
)

type WarehouseRepository interface {
	GetSites() ([]string, error)
	GetCriticalInventory(site string) ([]models.CriticalInventoryItem, error)
	GetCriticalInventoryPivot(site string) ([]models.CriticalInventoryPivotItem, []string, error)
}

type warehouseRepository struct {
	db *gorm.DB
}

func NewWarehouseRepository(db *gorm.DB) WarehouseRepository {
	return &warehouseRepository{db: db}
}

func (r *warehouseRepository) GetSites() ([]string, error) {
	var sites []string
	query := "SELECT `name` FROM sites WHERE status_value = 'Active' ORDER BY `name`"
	err := r.db.Raw(query).Scan(&sites).Error
	if err != nil {
		return nil, err
	}
	return sites, nil
}

func (r *warehouseRepository) GetCriticalInventory(site string) ([]models.CriticalInventoryItem, error) {
	site = strings.TrimSpace(site)

	query := `
SELECT site AS area, sku, prd AS namaPrdk, bln1, bln2, bln3, bln4, bln5, bln6, avg_ AS avg, stock, qty_po, po, po_date, estimasi, statue, note
FROM(
	SELECT s.name AS site, dt.sku
		, prd, bln1, bln2, bln3, bln4, bln5, bln6
		, bln1_+bln2_+bln3_+bln4_+bln5_+bln6_ AS jml
		, FLOOR((bln1+bln2+bln3+bln4+bln5+bln6) / (bln1_+bln2_+bln3_+bln4_+bln5_+bln6_))AS avg_
		, st.stock
		, CASE 
			WHEN st.stock < 0 THEN 'Check'
			WHEN (2*st.stock) < (FLOOR((bln1+bln2+bln3+bln4+bln5+bln6) / (bln1_+bln2_+bln3_+bln4_+bln5_+bln6_))) THEN 'UrgentReStock'
			WHEN st.stock < (FLOOR((bln1+bln2+bln3+bln4+bln5+bln6) / (bln1_+bln2_+bln3_+bln4_+bln5_+bln6_))) THEN 'LowStock'
			ELSE ''
		END AS note
		, po.qty AS qty_po, po.number AS po, po.createdAt AS po_date, po.estimasi, po.statue
	FROM(
		SELECT site_id
			, sku, prd, product_id
			, SUM(bln1)AS bln1
			, SUM(bln2)AS bln2
			, SUM(bln3)AS bln3
			, SUM(bln4)AS bln4
			, SUM(bln5)AS bln5
			, SUM(bln6)AS bln6
			, CASE WHEN SUM(bln1) > 0 THEN 1 ELSE 0 END AS bln1_
			, CASE WHEN SUM(bln2) > 0 THEN 1 ELSE 0 END AS bln2_
			, CASE WHEN SUM(bln3) > 0 THEN 1 ELSE 0 END AS bln3_
			, CASE WHEN SUM(bln4) > 0 THEN 1 ELSE 0 END AS bln4_
			, CASE WHEN SUM(bln5) > 0 THEN 1 ELSE 0 END AS bln5_
			, CASE WHEN SUM(bln6) > 0 THEN 1 ELSE 0 END AS bln6_
		FROM(
			SELECT CASE WHEN so.siteId = '00000001-b43e-11ee-bde1-bfcb6ff3c685' THEN '00000017-b43e-11ee-bde1-bfcb6ff3c685' ELSE so.siteId END AS site_id
				, f.createdAt, p.id AS product_id
				, p.sku, p.name AS prd, f.qty
				, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE() - INTERVAL 5 MONTH)) THEN f.qty ELSE 0 END AS bln1
				, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE() - INTERVAL 4 MONTH)) THEN f.qty ELSE 0 END AS bln2
				, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE() - INTERVAL 3 MONTH)) THEN f.qty ELSE 0 END AS bln3
				, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE() - INTERVAL 2 MONTH)) THEN f.qty ELSE 0 END AS bln4
				, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE() - INTERVAL 1 MONTH)) THEN f.qty ELSE 0 END AS bln5
				, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE())) THEN f.qty ELSE 0 END AS bln6
			FROM fulfillment_items f
			LEFT JOIN service_order_products sop ON sop.id = f.serviceOrderProductId
			LEFT JOIN service_orders so ON so.id = sop.serviceOrderId
			LEFT JOIN products p ON sop.productId = p.id
			WHERE 1
				AND DATE(f.createdAt) BETWEEN LAST_DAY(CURDATE() - INTERVAL 6 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
		)AS dt
		GROUP BY sku, site_id
	)AS dt
	LEFT JOIN sites s ON dt.site_id = s.id
	LEFT JOIN stocks st ON dt.product_id = st.productId AND dt.site_id = st.siteId
	LEFT JOIN(
		SELECT p.sku, po.number, po.createdAt, po.expectedDeliveryDate AS estimasi
			, poi.qty, poi.status_value AS statue
		FROM(
			SELECT productId, MAX(po)AS po, MAX(poi_id)AS poi_id
			FROM(
				SELECT po.number AS po
					, po.siteId
					, po.createdAt, po.expectedDeliveryDate
					, poi.productId, poi.id AS poi_id
				FROM purchase_orders po
				LEFT JOIN purchase_order_items poi ON po.id = poi.purchaseOrderId
				WHERE poi.deletedAt IS NULL
			)AS dt
			GROUP BY productId
		)AS dt
		LEFT JOIN purchase_orders po ON dt.po = po.number
		LEFT JOIN purchase_order_items poi ON dt.poi_id = poi.id
		LEFT JOIN products p ON dt.productId = p.id
		WHERE poi.deletedAt IS NULL
			AND po.createdAt >= LAST_DAY(CURDATE() - INTERVAL 1 MONTH )+ INTERVAL 1 DAY
	)AS po ON dt.sku = po.sku
	WHERE bln5_+bln6_ = 2 OR st.stock < 0
	ORDER BY note DESC, site_id, qty_po, sku, jml
)AS dt
WHERE stock < avg_
`

	var items []models.CriticalInventoryItem
	var err error

	if site != "" && strings.ToUpper(site) != "ALL" {
		query += " AND site = ? ORDER BY note DESC, qty_po, sku"
		err = r.db.Raw(query, site).Scan(&items).Error
	} else {
		query += " ORDER BY note DESC, qty_po, sku"
		err = r.db.Raw(query).Scan(&items).Error
	}

	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *warehouseRepository) GetCriticalInventoryPivot(site string) ([]models.CriticalInventoryPivotItem, []string, error) {
	sites, err := r.GetSites()
	if err != nil {
		sites = []string{}
	}

	critItems, err := r.GetCriticalInventory(site)
	if err != nil {
		return nil, sites, err
	}

	if len(critItems) == 0 {
		return []models.CriticalInventoryPivotItem{}, sites, nil
	}

	var skus []string
	skuMap := make(map[string]bool)
	critSiteMap := make(map[string][]string)
	noteMap := make(map[string]string)
	metaMap := make(map[string]models.CriticalInventoryItem)

	for _, it := range critItems {
		if !skuMap[it.Sku] {
			skuMap[it.Sku] = true
			skus = append(skus, it.Sku)
			metaMap[it.Sku] = it
		}
		critSiteMap[it.Sku] = append(critSiteMap[it.Sku], it.Area)

		currentNote := noteMap[it.Sku]
		if it.Note == "Check" {
			noteMap[it.Sku] = "Check"
		} else if it.Note == "UrgentReStock" && currentNote != "Check" {
			noteMap[it.Sku] = "UrgentReStock"
		} else if it.Note == "LowStock" && currentNote != "Check" && currentNote != "UrgentReStock" {
			noteMap[it.Sku] = "LowStock"
		}
	}

	// 1. Ambil stok real-time semua site aktif untuk SKU-SKU kritis
	type StockRow struct {
		Sku   string  `gorm:"column:sku"`
		Site  string  `gorm:"column:site"`
		Stock float64 `gorm:"column:stock"`
	}
	var stockRows []StockRow
	_ = r.db.Raw(`
		SELECT p.sku, s.name AS site, IFNULL(st.stock, 0) AS stock
		FROM stocks st
		JOIN products p ON st.productId = p.id
		JOIN sites s ON st.siteId = s.id
		WHERE s.status_value = 'Active' AND p.sku IN ?
	`, skus).Scan(&stockRows).Error

	stockMap := make(map[string]map[string]float64)
	for _, sr := range stockRows {
		if stockMap[sr.Sku] == nil {
			stockMap[sr.Sku] = make(map[string]float64)
		}
		stockMap[sr.Sku][sr.Site] = sr.Stock
	}

	// 2. Ambil average pemakaian semua site aktif untuk SKU-SKU kritis
	type AvgRow struct {
		Sku  string  `gorm:"column:sku"`
		Site string  `gorm:"column:site"`
		Avg  float64 `gorm:"column:avg"`
	}
	var avgRows []AvgRow
	_ = r.db.Raw(`
		SELECT dt.sku, s.name AS site
			, FLOOR((bln1+bln2+bln3+bln4+bln5+bln6) / (bln1_+bln2_+bln3_+bln4_+bln5_+bln6_)) AS avg
		FROM(
			SELECT site_id, sku
				, SUM(bln1)AS bln1, SUM(bln2)AS bln2, SUM(bln3)AS bln3
				, SUM(bln4)AS bln4, SUM(bln5)AS bln5, SUM(bln6)AS bln6
				, CASE WHEN SUM(bln1) > 0 THEN 1 ELSE 0 END AS bln1_
				, CASE WHEN SUM(bln2) > 0 THEN 1 ELSE 0 END AS bln2_
				, CASE WHEN SUM(bln3) > 0 THEN 1 ELSE 0 END AS bln3_
				, CASE WHEN SUM(bln4) > 0 THEN 1 ELSE 0 END AS bln4_
				, CASE WHEN SUM(bln5) > 0 THEN 1 ELSE 0 END AS bln5_
				, CASE WHEN SUM(bln6) > 0 THEN 1 ELSE 0 END AS bln6_
			FROM(
				SELECT CASE WHEN so.siteId = '00000001-b43e-11ee-bde1-bfcb6ff3c685' THEN '00000017-b43e-11ee-bde1-bfcb6ff3c685' ELSE so.siteId END AS site_id
					, f.createdAt, p.sku, f.qty
					, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE() - INTERVAL 5 MONTH)) THEN f.qty ELSE 0 END AS bln1
					, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE() - INTERVAL 4 MONTH)) THEN f.qty ELSE 0 END AS bln2
					, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE() - INTERVAL 3 MONTH)) THEN f.qty ELSE 0 END AS bln3
					, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE() - INTERVAL 2 MONTH)) THEN f.qty ELSE 0 END AS bln4
					, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE() - INTERVAL 1 MONTH)) THEN f.qty ELSE 0 END AS bln5
					, CASE WHEN MONTH(f.createdAt) = MONTH(LAST_DAY(CURDATE())) THEN f.qty ELSE 0 END AS bln6
				FROM fulfillment_items f
				LEFT JOIN service_order_products sop ON sop.id = f.serviceOrderProductId
				LEFT JOIN service_orders so ON so.id = sop.serviceOrderId
				LEFT JOIN products p ON sop.productId = p.id
				WHERE p.sku IN ? AND DATE(f.createdAt) BETWEEN LAST_DAY(CURDATE() - INTERVAL 6 MONTH) + INTERVAL 1 DAY AND LAST_DAY(CURDATE())
			)AS dt
			GROUP BY sku, site_id
		)AS dt
		JOIN sites s ON dt.site_id = s.id
		WHERE s.status_value = 'Active'
	`, skus).Scan(&avgRows).Error

	avgMap := make(map[string]map[string]float64)
	for _, ar := range avgRows {
		if avgMap[ar.Sku] == nil {
			avgMap[ar.Sku] = make(map[string]float64)
		}
		avgMap[ar.Sku][ar.Site] = ar.Avg
	}

	// 3. Bangun data pivot per SKU
	var pivotItems []models.CriticalInventoryPivotItem
	for _, sku := range skus {
		meta := metaMap[sku]
		sitesData := make(map[string]models.SiteStockData)
		var totAvg, totStock float64

		for _, siteName := range sites {
			var stVal, avgVal float64
			if m, ok := stockMap[sku]; ok {
				stVal = m[siteName]
			}
			if m, ok := avgMap[sku]; ok {
				avgVal = m[siteName]
			}
			sitesData[siteName] = models.SiteStockData{Avg: avgVal, Stock: stVal}
			totAvg += avgVal
			totStock += stVal
		}

		item := models.CriticalInventoryPivotItem{
			Sku:           meta.Sku,
			NamaProduk:    meta.NamaPrdk,
			Sites:         sitesData,
			TotalAvg:      totAvg,
			TotalStock:    totStock,
			Note:          noteMap[sku],
			CriticalSites: critSiteMap[sku],
			QtyPo:         meta.QtyPo,
			Po:            meta.Po,
			PoDate:        meta.PoDate,
			Estimasi:      meta.Estimasi,
			Statue:        meta.Statue,
		}
		pivotItems = append(pivotItems, item)
	}

	return pivotItems, sites, nil
}
