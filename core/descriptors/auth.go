package descriptors

import (
	"time"
)

const UserTableName = "__users"

const (
	RoleSa    = "sa"
	RoleAdmin = "admin"
	RoleUser  = "user"
	RoleGuest = "guest"
)

type Role struct {
	Id              int64  `json:"id" mapstructure:"id"`
	Name            string `json:"name" mapstructure:"name"`
	Disabled        bool   `json:"disabled" mapstructure:"disabled"`
	DashboardPageId string `json:"dashboardPageId" mapstructure:"dashboard_page_id"`
	MenuId          string `json:"menuId" mapstructure:"menu_id"`
}

type User struct {
	Id            int64     `json:"id" mapstructure:"id"`
	Email         string    `json:"email" mapstructure:"email"`
	PasswordHash  string    `json:"-" mapstructure:"password_hash"`
	Roles         []string  `json:"roles" mapstructure:"roles"`
	RolesDetails  []Role    `json:"rolesDetails,omitempty"`
	DefaultRoleId *int64    `json:"defaultRoleId,omitempty" mapstructure:"default_role_id"`
	AvatarPath    string         `json:"avatarPath" mapstructure:"avatar_path"`
	Channels      []UserChannel  `json:"channels,omitempty" mapstructure:"channels"`
	CreatedAt     time.Time      `json:"createdAt" mapstructure:"created_at"`
	UpdatedAt     time.Time      `json:"updatedAt" mapstructure:"updated_at"`
}
