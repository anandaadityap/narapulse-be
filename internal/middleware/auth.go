package middleware

import (
	"narapulse-be/internal/config"
	entity "narapulse-be/internal/models/entity"
	"narapulse-be/internal/pkg/utils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware validates JWT token
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return entity.UnauthorizedResponse(c, "Authorization header is required")
		}

		// Check if header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return entity.UnauthorizedResponse(c, "Invalid authorization header format")
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			return entity.UnauthorizedResponse(c, "Token is required")
		}

		// Validate token
		cfg := config.Load()
		claims, err := utils.ValidateToken(token, cfg.JWTSecret)
		if err != nil {
			return entity.UnauthorizedResponse(c, "Invalid or expired token")
		}

		// Store user info in context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)

		return c.Next()
	}
}

// AdminMiddleware checks if user has admin role
func AdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("user_role")
		if userRole == nil {
			return entity.UnauthorizedResponse(c, "User role not found")
		}

		role, ok := userRole.(string)
		if !ok || role != "admin" {
			return entity.ForbiddenResponse(c, "Admin access required")
		}

		return c.Next()
	}
}

// GetUserIDFromContext extracts user ID from fiber context
func GetUserIDFromContext(c *fiber.Ctx) (uint, error) {
	userID := c.Locals("user_id")
	if userID == nil {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "User ID not found in context")
	}

	id, ok := userID.(uint)
	if !ok {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "Invalid user ID format")
	}

	return id, nil
}

// GetUserRoleFromContext extracts user role from fiber context
func GetUserRoleFromContext(c *fiber.Ctx) (string, error) {
	userRole := c.Locals("user_role")
	if userRole == nil {
		return "", fiber.NewError(fiber.StatusUnauthorized, "User role not found in context")
	}

	role, ok := userRole.(string)
	if !ok {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Invalid user role format")
	}

	return role, nil
}