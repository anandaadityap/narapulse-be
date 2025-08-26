package services

import (
	"log"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

type CasbinService struct {
	enforcer *casbin.Enforcer
}

func NewCasbinService(db *gorm.DB) (*CasbinService, error) {
	// Initialize the adapter
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}

	// Initialize the enforcer
	enforcer, err := casbin.NewEnforcer("configs/rbac_model.conf", adapter)
	if err != nil {
		return nil, err
	}

	// Load policy from database
	err = enforcer.LoadPolicy()
	if err != nil {
		return nil, err
	}

	// Load initial policies if database is empty
	policies, err := enforcer.GetGroupingPolicy()
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		err = loadInitialPolicies(enforcer)
		if err != nil {
			return nil, err
		}
	}

	log.Println("Casbin enforcer initialized successfully")
	return &CasbinService{enforcer: enforcer}, nil
}

func loadInitialPolicies(enforcer *casbin.Enforcer) error {
	// Add role-based policies
	policies := [][]string{
		{"admin", "/api/v1/admin/*", "*"},
		{"admin", "/api/v1/profile", "*"},
		{"user", "/api/v1/profile", "GET"},
		{"user", "/api/v1/profile", "PUT"},
	}

	for _, policy := range policies {
		_, err := enforcer.AddPolicy(policy)
		if err != nil {
			return err
		}
	}

	// Add role assignments
	roleAssignments := [][]string{
		{"admin@narapulse.com", "admin"},
	}

	for _, assignment := range roleAssignments {
		_, err := enforcer.AddRoleForUser(assignment[0], assignment[1])
		if err != nil {
			return err
		}
	}

	// Save policies to database
	return enforcer.SavePolicy()
}

// Enforce checks if a user has permission to access a resource
func (cs *CasbinService) Enforce(user, resource, action string) (bool, error) {
	return cs.enforcer.Enforce(user, resource, action)
}

// AddPolicy adds a new policy
func (cs *CasbinService) AddPolicy(role, resource, action string) (bool, error) {
	return cs.enforcer.AddPolicy(role, resource, action)
}

// RemovePolicy removes a policy
func (cs *CasbinService) RemovePolicy(role, resource, action string) (bool, error) {
	return cs.enforcer.RemovePolicy(role, resource, action)
}

// AddRoleForUser assigns a role to a user
func (cs *CasbinService) AddRoleForUser(user, role string) (bool, error) {
	return cs.enforcer.AddRoleForUser(user, role)
}

// DeleteRoleForUser removes a role from a user
func (cs *CasbinService) DeleteRoleForUser(user, role string) (bool, error) {
	return cs.enforcer.DeleteRoleForUser(user, role)
}

// GetRolesForUser gets all roles for a user
func (cs *CasbinService) GetRolesForUser(user string) ([]string, error) {
	roles, err := cs.enforcer.GetRolesForUser(user)
	return roles, err
}

// GetUsersForRole gets all users with a specific role
func (cs *CasbinService) GetUsersForRole(role string) ([]string, error) {
	users, err := cs.enforcer.GetUsersForRole(role)
	return users, err
}