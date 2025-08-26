package handlers

import (
	"narapulse-be/internal/config"
	entity "narapulse-be/internal/models/entity"
	"narapulse-be/internal/repositories"
	"narapulse-be/internal/services"
	"narapulse-be/internal/pkg/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type AuthHandler struct {
	userService services.UserService
	validator   *validator.Validate
	config      *config.Config
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	userRepo := repositories.NewUserRepository(db)
	userService := services.NewUserService(userRepo)
	return &AuthHandler{
		userService: userService,
		validator:   validator.New(),
		config:      config.Load(),
	}
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with email, username, and password
// @Tags auth
// @Accept json
// @Produce json
// @Param user body models.UserCreateRequest true "User registration data"
// @Success 201 {object} models.StandardResponse{data=models.UserResponse}
// @Failure 400 {object} models.StandardResponse
// @Failure 500 {object} models.StandardResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req entity.UserCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return entity.BadRequestResponse(c, "Invalid request body", err.Error())
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return entity.BadRequestResponse(c, "Validation failed", err.Error())
	}

	// Create user
	user, err := h.userService.CreateUser(&req)
	if err != nil {
		return entity.BadRequestResponse(c, "Failed to create user", err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(entity.StandardResponse{
		Success: true,
		Message: "User registered successfully",
		Data:    user,
	})
}

// Login godoc
// @Summary Login user
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.StandardResponse{data=models.LoginResponse}
// @Failure 400 {object} models.StandardResponse
// @Failure 401 {object} models.StandardResponse
// @Failure 500 {object} models.StandardResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req entity.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return entity.BadRequestResponse(c, "Invalid request body", err.Error())
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return entity.BadRequestResponse(c, "Validation failed", err.Error())
	}

	// Authenticate user
	user, err := h.userService.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		return entity.UnauthorizedResponse(c, err.Error())
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, user.Email, user.Role, h.config.JWTSecret)
	if err != nil {
		return entity.InternalServerErrorResponse(c, "Failed to generate token", err.Error())
	}

	// Convert user to response format
	userResponse := &entity.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	response := entity.LoginResponse{
		Token: token,
		User:  *userResponse,
	}

	return entity.SuccessResponse(c, "Login successful", response)
}