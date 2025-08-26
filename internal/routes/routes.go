package routes

import (
	_ "narapulse-be/docs"
	"narapulse-be/internal/handlers"
	"narapulse-be/internal/middleware"
	"narapulse-be/internal/repositories"
	"narapulse-be/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/swaggo/fiber-swagger"
	"gorm.io/gorm"
)

func Setup(app *fiber.App, db *gorm.DB) {
	// Initialize repositories
	dataSourceRepo := repositories.NewDataSourceRepository(db)
	schemaRepo := repositories.NewSchemaRepository(db)

	// Initialize services
	connectorService := services.NewConnectorService()
	dataSourceService := services.NewDataSourceService(dataSourceRepo, schemaRepo, connectorService)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(db)
	authHandler := handlers.NewAuthHandler(db)
	// Initialize DataSourceHandler
	dataSourceHandler := handlers.NewDataSourceHandler(dataSourceService)

	// API routes
	api := app.Group("/api/v1")

	// Public routes
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// Protected routes
	protected := api.Group("/", middleware.AuthMiddleware())
	protected.Get("/profile", userHandler.GetProfile)
	protected.Put("/profile", userHandler.UpdateProfile)

	// Data Sources routes (protected)
	dataSources := protected.Group("/data-sources")
	dataSources.Post("/", dataSourceHandler.CreateDataSource)
	dataSources.Get("/", dataSourceHandler.GetDataSources)
	dataSources.Get("/:id", dataSourceHandler.GetDataSource)
	dataSources.Put("/:id", dataSourceHandler.UpdateDataSource)
	dataSources.Delete("/:id", dataSourceHandler.DeleteDataSource)
	dataSources.Post("/test-connection", dataSourceHandler.TestConnection)
	dataSources.Post("/:id/refresh-schema", dataSourceHandler.RefreshSchema)
	dataSources.Post("/upload", dataSourceHandler.UploadFile)

	// Admin routes
	admin := api.Group("/admin", middleware.AuthMiddleware(), middleware.AdminMiddleware())
	admin.Get("/users", userHandler.GetAllUsers)
	admin.Delete("/users/:id", userHandler.DeleteUser)

	// Swagger documentation
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Server is running",
			"data":    nil,
		})
	})
}