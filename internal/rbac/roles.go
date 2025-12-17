package rbac

// Role names. Keep these stable; they are part of auth/RBAC contracts.
const (
	RoleOwner           = "owner"
	RoleAgent           = "agent"
	RoleAnalyst         = "analyst"
	RoleFinance         = "finance"
	RoleSuperAdmin      = "super_admin"
	RoleNetworkOperator = "network_operator" // hidden role
)

func IsSuperAdmin(role string) bool { return role == RoleSuperAdmin }

func IsHiddenRole(role string) bool { return role == RoleNetworkOperator }
