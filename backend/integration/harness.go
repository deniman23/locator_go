package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"locator/config/messaging"
	"locator/controllers"
	"locator/dao"
	"locator/models"
	"locator/router"
	"locator/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestEnv holds a wired Gin engine and DB for HTTP integration tests.
type TestEnv struct {
	DB        *gorm.DB
	Router    *gin.Engine
	Admin     models.User
	Device    models.User
	AdminKey  string
	DeviceKey string
}

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("set INTEGRATION_TEST=1 and Postgres to run integration tests")
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	host := envOr("DB_HOST", "127.0.0.1")
	port := envOr("DB_PORT", "5433")
	user := envOr("DB_USER", "locator_user")
	pass := envOr("DB_PASSWORD", "change_me")
	// Never default to the production DB name — harness TRUNCATEs all tables.
	name := envOr("DB_NAME", "locator_db_test")
	ssl := envOr("DB_SSLMODE", "disable")

	if name == "locator_db" && os.Getenv("ALLOW_PROD_DB_WIPE") != "1" {
		t.Fatalf("refusing to run integration tests against DB_NAME=%q (production). "+
			"Use DB_NAME=locator_db_test (or set ALLOW_PROD_DB_WIPE=1 only for disposable DBs)", name)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, ssl)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("postgres unavailable (%v); create DB %q and start docker compose db", err, name)
	}
	return db
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func setupEnv(t *testing.T) *TestEnv {
	t.Helper()
	requireIntegration(t)
	gin.SetMode(gin.TestMode)

	db := openTestDB(t)
	if err := db.AutoMigrate(
		&models.User{},
		&models.Location{},
		&models.LocationRequest{},
		&models.DeviceCommand{},
		&models.DeviceReport{},
		&models.Checkpoint{},
		&models.Visit{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Isolate each test run: wipe domain tables (keep schema).
	for _, table := range []string{
		"visits", "locations", "location_requests", "device_commands", "device_reports", "checkpoints", "users",
	} {
		_ = db.Exec("TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE").Error
	}

	wd, _ := os.Getwd()
	// Ensure QR dir exists relative to backend or repo root when tests run from integration/
	qrDir := filepath.Join(wd, "..", "static", "qrcode")
	_ = os.MkdirAll(qrDir, 0o755)
	_ = os.MkdirAll(filepath.Join(wd, "..", "static", "releases"), 0o755)

	adminKey := "integration-admin-key-001"
	deviceKey := "integration-device-key-001"
	adminHash, err := bcrypt.GenerateFromPassword([]byte(adminKey), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	deviceHash, err := bcrypt.GenerateFromPassword([]byte(deviceKey), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	admin := models.User{Name: "it-admin", ApiKey: string(adminHash), IsAdmin: true}
	device := models.User{Name: "it-device", ApiKey: string(deviceHash), IsAdmin: false}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	noopPub := &messaging.Publisher{} // nil channel → no-op publish

	locationDAO := dao.NewLocationDAO(db)
	locationService := service.NewLocationService(locationDAO)
	locationRequestDAO := dao.NewLocationRequestDAO(db)
	locationRequestService := service.NewLocationRequestService(locationRequestDAO)
	deviceCommandDAO := dao.NewDeviceCommandDAO(db)
	deviceReportDAO := dao.NewDeviceReportDAO(db)
	deviceCommandService := service.NewDeviceCommandService(deviceCommandDAO, locationRequestService)
	deviceReportService := service.NewDeviceReportService(deviceReportDAO)
	deviceStatusService := service.NewDeviceStatusService(locationDAO, deviceReportDAO)

	baseURL := "http://localhost:8080"
	appReleaseController := controllers.NewAppReleaseController(
		filepath.Join(wd, "..", "static", "releases", "manifest.json"),
		filepath.Join(wd, "..", "static", "releases"),
		baseURL,
	)
	deviceController := controllers.NewDeviceController(
		deviceCommandService, deviceReportService, deviceStatusService, locationRequestService, appReleaseController,
	)
	locationRequestController := controllers.NewLocationRequestController(locationRequestService, deviceCommandService)

	checkpointDAO := dao.NewCheckpointDAO(db)
	checkpointService := service.NewCheckpointService(checkpointDAO)
	visitDAO := dao.NewVisitDAO(db)
	travelSegmentService := service.NewTravelSegmentService(locationDAO, checkpointService)
	visitService := service.NewVisitService(visitDAO, travelSegmentService)
	locationController := controllers.NewLocationController(
		locationService, locationRequestService, deviceCommandService, noopPub, "",
	)
	checkpointController := controllers.NewCheckpointController(
		checkpointService, locationService, visitService, noopPub,
	)
	visitController := controllers.NewVisitController(visitService)
	eventController := controllers.NewEventController(noopPub)

	userDAO := dao.NewUserDAO(db)
	userService := service.NewUserService(userDAO)
	userController := controllers.NewUserController(userService, deviceCommandService)

	r := router.InitRoutes(
		locationController,
		locationRequestController,
		deviceController,
		appReleaseController,
		checkpointController,
		visitController,
		eventController,
		userController,
		userService,
	)

	return &TestEnv{
		DB:        db,
		Router:    r,
		Admin:     admin,
		Device:    device,
		AdminKey:  adminKey,
		DeviceKey: deviceKey,
	}
}
