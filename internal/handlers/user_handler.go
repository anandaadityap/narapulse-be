package handlers

import (
	"narapulse-be/internal/middleware"
	entity "narapulse-be/internal/models/entity"
	"narapulse-be/internal/repositories"
	"narapulse-be/internal/services"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserHandler struct {
	userService services.UserService
	validator   *validator.Validate
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	userRepo := repositories.NewUserRepository(db)
	userService := services.NewUserService(userRepo)
	return &UserHandler{
		userService: userService,
		validator:   validator.New(),
	}
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get the profile of the authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entity.StandardResponse{data=entity.UserResponse}
// @Failure 401 {object} entity.StandardResponse
// @Failure 404 {object} entity.StandardResponse
// @Failure 500 {object} entity.StandardResponse
// @Router /profile [get]
func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		return entity.UnauthorizedResponse(c, err.Error())
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		return entity.NotFoundResponse(c, err.Error())
	}

	return entity.SuccessResponse(c, "Profile retrieved successfully", user)
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update the profile of the authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body entity.UserUpdateRequest true "User update data"
// @Success 200 {object} entity.StandardResponse{data=entity.UserResponse}
// @Failure 400 {object} entity.StandardResponse
// @Failure 401 {object} entity.StandardResponse
// @Failure 404 {object} entity.StandardResponse
// @Failure 500 {object} entity.StandardResponse
// @Router /profile [put]
func (h *UserHandler) UpdateProfile(c *fiber.Ctx) error {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		return entity.UnauthorizedResponse(c, err.Error())
	}

	var req entity.UserUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return entity.BadRequestResponse(c, "Invalid request body", err.Error())
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return entity.BadRequestResponse(c, "Validation failed", err.Error())
	}

	user, err := h.userService.UpdateUser(userID, &req)
	if err != nil {
		return entity.BadRequestResponse(c, "Failed to update profile", err.Error())
	}

	return entity.SuccessResponse(c, "Profile updated successfully", user)
}

// GetAllUsers godoc
// @Summary Get all users (Admin only)
// @Description Get a paginated list of all users
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} entity.StandardResponse{data=[]entity.UserResponse}
// @Failure 401 {object} entity.StandardResponse
// @Failure 403 {object} entity.StandardResponse
// @Failure 500 {object} entity.StandardResponse
// @Router /admin/users [get]
func (h *UserHandler) GetAllUsers(c *fiber.Ctx) error {
	users, err := h.userService.GetAllUsers()
	if err != nil {
		return entity.InternalServerErrorResponse(c, "Failed to retrieve users", err.Error())
	}

	return entity.SuccessResponse(c, "Users retrieved successfully", users)
}

// DeleteUser godoc
// @Summary Delete user (Admin only)
// @Description Delete a user by ID
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} entity.StandardResponse
// @Failure 400 {object} entity.StandardResponse
// @Failure 401 {object} entity.StandardResponse
// @Failure 403 {object} entity.StandardResponse
// @Failure 404 {object} entity.StandardResponse
// @Failure 500 {object} entity.StandardResponse
// @Router /admin/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	userIDStr := c.Params("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return entity.BadRequestResponse(c, "Invalid user ID", err.Error())
	}

	if err := h.userService.DeleteUser(uint(userID)); err != nil {
		return entity.BadRequestResponse(c, "Failed to delete user", err.Error())
	}

	return entity.SuccessResponse(c, "User deleted successfully", nil)
}