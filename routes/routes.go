package routes

import (
	"net/http"

	"asm/handler"
	"asm/repository"
	"asm/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(dbCsv *gorm.DB, dbHo *gorm.DB) *gin.Engine {
	r := gin.Default()

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Static Assets
	r.Static("/web", "./web")
	r.Static("/components", "./web/components")
	r.StaticFile("/favicon.ico", "./web/images/logo-sar.png")

	// Auth Handler (diinisialisasi lebih awal agar bisa memproteksi halaman web)
	var authHandler *handler.AuthHandler
	if dbCsv != nil {
		authRepo := repository.NewAuthRepository(dbCsv)
		authSvc := service.NewAuthService(authRepo)
		authHandler = handler.NewAuthHandler(authSvc)
	}

	// Public Auth Page
	r.StaticFile("/login", "./web/login.html")

	// Root Route: Redirects to /prj-csv
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/prj-csv")
	})

	// Protected Web Pages (Wajib punya session login, jika tidak redirect ke /login)
	pages := r.Group("/")
	if authHandler != nil {
		pages.Use(authHandler.RequirePageAuth())
	}
	{
		// PRJ-HO Routes (Dashboard dari project HO)
		pages.StaticFile("/prj-ho", "./web/prj-ho/operation-dashboard.html")
		pages.StaticFile("/prj-ho/", "./web/prj-ho/operation-dashboard.html")
		pages.StaticFile("/prj-ho/dashboard", "./web/prj-ho/operation-dashboard.html")
		pages.StaticFile("/prj-ho/operation-dashboard", "./web/prj-ho/operation-dashboard.html")
		pages.StaticFile("/prj-ho/spk-per-cust", "./web/prj-ho/spk-per-cust.html")
		pages.StaticFile("/prj-ho/warehouse/critical-inventory", "./web/prj-ho/critical-inventory.html")

		// Backwards compatibility for root paths
		pages.StaticFile("/dashboard", "./web/prj-ho/operation-dashboard.html")
		pages.StaticFile("/operation-dashboard", "./web/prj-ho/operation-dashboard.html")
		pages.StaticFile("/spk-per-cust", "./web/prj-ho/spk-per-cust.html")
		pages.StaticFile("/warehouse/critical-inventory", "./web/prj-ho/critical-inventory.html")

		// PRJ-CSV Routes (Dashboard project CSV)
		pages.StaticFile("/prj-csv", "./web/prj-csv/index.html")
		pages.StaticFile("/prj-csv/", "./web/prj-csv/index.html")
		pages.StaticFile("/prj-csv/dashboard", "./web/prj-csv/index.html")
		pages.StaticFile("/prj-csv/spk-per-cust", "./web/prj-csv/spk-per-cust.html")
		pages.StaticFile("/prj-csv/spk-slash-cust", "./web/prj-csv/spk-slash-cust.html")
		pages.StaticFile("/prj-csv/spk-cust", "./web/prj-csv/spk-slash-cust.html")
		pages.StaticFile("/prj-csv/summary-per-cust", "./web/prj-csv/summary-per-cust.html")
		pages.StaticFile("/prj-csv/resume-per-kategori", "./web/prj-csv/resume-per-kategori.html")
		pages.StaticFile("/prj-csv/resume-per-kategori/detail", "./web/prj-csv/resume-per-kategori-detail.html")
		pages.StaticFile("/prj-csv/resume-per-kategori-detail", "./web/prj-csv/resume-per-kategori-detail.html")
		pages.StaticFile("/prj-csv/spk-detail", "./web/prj-csv/spk-detail.html")
		pages.StaticFile("/prj-csv/produksi/resume-per-hari", "./web/prj-csv/produksi/resume-per-hari.html")
		pages.StaticFile("/prj-csv/resume-per-hari", "./web/prj-csv/produksi/resume-per-hari.html")
		pages.StaticFile("/prj-csv/produksi/summary-per-driver", "./web/prj-csv/produksi/summary-per-driver.html")
		pages.StaticFile("/prj-csv/summary-per-driver", "./web/prj-csv/produksi/summary-per-driver.html")
		pages.StaticFile("/prj-csv/produksi/monitoring-ritase", "./web/prj-csv/produksi/monitoring-ritase.html")
		pages.StaticFile("/prj-csv/monitoring-ritase", "./web/prj-csv/produksi/monitoring-ritase.html")
		pages.StaticFile("/prj-csv/produksi/transfer-out", "./web/prj-csv/produksi/transfer-out.html")
		pages.StaticFile("/prj-csv/transfer-out", "./web/prj-csv/produksi/transfer-out.html")
		pages.StaticFile("/prj-csv/master-data/prior-unit", "./web/prj-csv/master-data/prior-unit.html")
		pages.StaticFile("/prj-csv/prior-unit", "./web/prj-csv/master-data/prior-unit.html")
		pages.StaticFile("/prj-csv/master-data/closing-finance", "./web/prj-csv/master-data/closing-finance.html")
		pages.StaticFile("/prj-csv/closing-finance", "./web/prj-csv/master-data/closing-finance.html")
		pages.StaticFile("/prj-csv/colokan-solar/pemakaian-solar", "./web/prj-csv/colokan-solar/pemakaian-solar.html")
		pages.StaticFile("/prj-csv/pemakaian-solar", "./web/prj-csv/colokan-solar/pemakaian-solar.html")
		pages.StaticFile("/colokan-solar/pemakaian-solar", "./web/prj-csv/colokan-solar/pemakaian-solar.html")

		// Administrator Routes
		pages.StaticFile("/admin/users", "./web/admin/users.html")
		pages.StaticFile("/admin/groups", "./web/admin/groups.html")
	}

	// HO Repositories, Services, and Handlers
	var hoDashboardHandler *handler.HoDashboardHandler
	var operationHandler *handler.OperationHandler
	var spkHandler *handler.SpkHandler
	var warehouseHandler *handler.WarehouseHandler

	if dbHo != nil {
		hoDashboardRepo := repository.NewDashboardRepository(dbHo)
		operationRepo := repository.NewOperationRepository(dbHo)
		spkRepo := repository.NewSpkRepository(dbHo)
		warehouseRepo := repository.NewWarehouseRepository(dbHo)

		hoDashboardSvc := service.NewDashboardService(hoDashboardRepo)
		operationSvc := service.NewOperationService(operationRepo)
		spkSvc := service.NewSpkService(spkRepo)
		warehouseSvc := service.NewWarehouseService(warehouseRepo)

		hoDashboardHandler = handler.NewHoDashboardHandler(hoDashboardSvc)
		operationHandler = handler.NewOperationHandler(operationSvc)
		spkHandler = handler.NewSpkHandler(spkSvc)
		warehouseHandler = handler.NewWarehouseHandler(warehouseSvc)
	}

	// CSV Project Handlers
	var csvHandler *handler.CsvDashboardHandler
	var csvSpkHandler *handler.CsvSpkHandler
	var produksiHandler *handler.ProduksiHandler
	var priorUnitHandler *handler.PriorUnitHandler
	var closingFinanceHandler *handler.ClosingFinanceHandler
	var solarHandler *handler.SolarHandler
	var stockpileSyncHandler *handler.StockpileSyncHandler

	csvSyncSvc := service.NewCsvSyncService(dbCsv)
	csvSyncHandler := handler.NewCsvSyncHandler(csvSyncSvc)

	if dbCsv != nil {
		csvRepo := repository.NewCsvDashboardRepository(dbCsv)
		csvSvc := service.NewCsvDashboardService(csvRepo)
		csvHandler = handler.NewCsvDashboardHandler(csvSvc)

		csvSpkRepo := repository.NewCsvSpkRepository(dbCsv)
		csvSpkSvc := service.NewCsvSpkService(csvSpkRepo)
		csvSpkHandler = handler.NewCsvSpkHandler(csvSpkSvc)

		produksiRepo := repository.NewProduksiRepository(dbCsv)
		produksiSvc := service.NewProduksiService(produksiRepo)
		produksiHandler = handler.NewProduksiHandler(produksiSvc)

		priorUnitRepo := repository.NewPriorUnitRepository(dbCsv)
		priorUnitSvc := service.NewPriorUnitService(priorUnitRepo)
		priorUnitHandler = handler.NewPriorUnitHandler(priorUnitSvc)

		closingFinanceRepo := repository.NewClosingFinanceRepository(dbCsv)
		closingFinanceSvc := service.NewClosingFinanceService(closingFinanceRepo)
		closingFinanceHandler = handler.NewClosingFinanceHandler(closingFinanceSvc)

		solarRepo := repository.NewSolarRepository(dbCsv)
		solarSyncSvc := service.NewSolarSyncService(dbCsv)
		solarSvc := service.NewSolarService(solarRepo)
		solarHandler = handler.NewSolarHandler(solarSvc, solarSyncSvc)


		stockpileSyncSvc := service.NewStockpileSyncService(dbCsv)
		stockpileSyncHandler = handler.NewStockpileSyncHandler(stockpileSyncSvc)
	}

	// API Endpoints
	api := r.Group("/api/v1")
	{
		// 1. HO Operation Dashboard, SPK, and Warehouse Endpoints
		if operationHandler != nil {
			api.GET("/dashboard/operation", operationHandler.GetOperationSummary)
		} else {
			api.GET("/dashboard/operation", func(c *gin.Context) {
				c.JSON(500, gin.H{"status": "error", "message": "Database HO (10.2.1.2) belum terhubung"})
			})
		}

		if hoDashboardHandler != nil {
			api.GET("/dashboard/visitors", hoDashboardHandler.GetVisitorSummary)
		}

		if spkHandler != nil {
			api.GET("/reports/spk-per-cust", spkHandler.GetSpkPerCust)
			api.GET("/reports/spk-filter-options", spkHandler.GetFilterOptions)
		}

		if warehouseHandler != nil {
			api.GET("/ho/warehouse/critical-inventory", warehouseHandler.GetCriticalInventory)
			api.GET("/ho/warehouse/sites", warehouseHandler.GetSites)
		}

		// 2. CSV Dashboard, SPK, and Produksi API
		if csvHandler != nil {
			api.GET("/csv/dashboard/visitors", csvHandler.GetVisitorSummary)
			api.GET("/csv/dashboard/areas", csvHandler.GetAreas)
			api.GET("/csv/dashboard/categories", csvHandler.GetCategoryDashboardData)
		}

		if csvSpkHandler != nil {
			api.GET("/csv/reports/spk-filter-options", csvSpkHandler.GetFilterOptions)
			api.GET("/csv/reports/spk-per-cust", csvSpkHandler.GetSpkPerCust)
			api.GET("/csv/reports/summary-per-cust", csvSpkHandler.GetSummaryPerCust)
			api.GET("/csv/reports/resume-per-kategori", csvSpkHandler.GetResumePerKategori)
			api.GET("/csv/reports/resume-per-kategori/detail", csvSpkHandler.GetResumeKategoriDetail)
			api.GET("/csv/reports/spk-detail", csvSpkHandler.GetSpkDetail)
		}

		if produksiHandler != nil {
			api.GET("/csv/produksi/resume-per-hari", produksiHandler.GetResumePerHari)
			api.GET("/csv/produksi/summary-per-driver", produksiHandler.GetSummaryPerDriver)
			api.GET("/csv/produksi/monitoring-ritase", produksiHandler.GetMonitoringRitase)
			api.GET("/csv/produksi/transfer-out", produksiHandler.GetTransferOut)
			api.GET("/csv/produksi/transfer-out/filters", produksiHandler.GetTransferOutFilterOptions)
		}

		// Master Data: Prior Unit CRUD Endpoints
		if priorUnitHandler != nil {
			api.GET("/csv/master/prior-unit", priorUnitHandler.GetAll)
			api.GET("/csv/master/prior-unit/:id", priorUnitHandler.GetByID)
			api.POST("/csv/master/prior-unit", priorUnitHandler.Create)
			api.PUT("/csv/master/prior-unit/:id", priorUnitHandler.Update)
			api.DELETE("/csv/master/prior-unit/:id", priorUnitHandler.Delete)
		}

		// Master Data: Closing Finance Endpoints
		if closingFinanceHandler != nil {
			api.GET("/csv/master/closing-finance", closingFinanceHandler.GetData)
			api.GET("/csv/master/closing-finance/filters", closingFinanceHandler.GetFilterOptions)
			api.POST("/csv/master/closing-finance/upload", closingFinanceHandler.UploadExcel)
			api.DELETE("/csv/master/closing-finance/truncate", closingFinanceHandler.Truncate)
		}

		// Colokan Solar Endpoints
		if solarHandler != nil {
			api.GET("/solar/pemakaian", solarHandler.GetPemakaianSolar)
			api.GET("/solar/filter-options", solarHandler.GetFilterOptions)
			api.POST("/solar/sync", solarHandler.TriggerSync)
			api.GET("/solar/sync/status", solarHandler.GetSyncStatus)
		}

		// Auth & Administrator Endpoints
		if authHandler != nil {
			api.POST("/auth/login", authHandler.Login)
			api.POST("/auth/logout", authHandler.Logout)
			api.GET("/auth/me", authHandler.GetMe)
			api.POST("/auth/change-password", authHandler.RequireAuth(), authHandler.ChangePassword)

			// Master User
			api.GET("/admin/users", authHandler.RequireAuth(), authHandler.GetAllUsers)
			api.POST("/admin/users", authHandler.RequireAuth(), authHandler.CreateUser)
			api.PUT("/admin/users/:id", authHandler.RequireAuth(), authHandler.UpdateUser)
			api.DELETE("/admin/users/:id", authHandler.RequireAuth(), authHandler.DeleteUser)

			// Master Group & Setup Menu
			api.GET("/admin/groups", authHandler.RequireAuth(), authHandler.GetAllGroups)
			api.POST("/admin/groups", authHandler.RequireAuth(), authHandler.CreateGroup)
			api.PUT("/admin/groups/:id", authHandler.RequireAuth(), authHandler.UpdateGroup)
			api.DELETE("/admin/groups/:id", authHandler.RequireAuth(), authHandler.DeleteGroup)
			api.GET("/admin/groups/:id/menus", authHandler.RequireAuth(), authHandler.GetGroupMenus)
			api.POST("/admin/groups/:id/menus", authHandler.RequireAuth(), authHandler.SaveGroupMenus)
		}

		// CSV Background Sync Endpoints
		api.POST("/csv/sync", csvSyncHandler.TriggerSync)
		api.GET("/csv/sync/status", csvSyncHandler.GetSyncStatus)

		// Stockpile Sync Endpoints (Metabase -> api_stockpile)
		if stockpileSyncHandler != nil {
			api.POST("/stockpile/sync", stockpileSyncHandler.TriggerSync)
			api.GET("/stockpile/sync/status", stockpileSyncHandler.GetSyncStatus)
		}
	}

	return r
}
