package services

import (
	"context"
	"os"
	"testing"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/stretchr/testify/assert"
)

func TestBootstrapAdmin(t *testing.T) {
	tests := []struct {
		name           string
		defaultEmail   string
		defaultPass    string
		isTestEnv      bool
		setupEnv       func()
		cleanupEnv     func()
		preExistingUser *descriptors.User
		expectedEmail  string
		expectedRoles  []string
		expectCreated  bool
	}{
		{
			name:          "Auto-seed in test env",
			defaultEmail:  "",
			defaultPass:   "",
			isTestEnv:     true,
			setupEnv:      func() {},
			cleanupEnv:    func() {},
			expectedEmail: "admin@aigen.local",
			expectedRoles: []string{descriptors.RoleSa, descriptors.RoleAdmin, descriptors.RoleUser},
			expectCreated: true,
		},
		{
			name:         "Override via environment variables",
			defaultEmail: "default@aigen.local",
			defaultPass:  "defaultpass",
			isTestEnv:    false,
			setupEnv: func() {
				os.Setenv("AIGEN_ADMIN_EMAIL", "custom@aigen.local")
				os.Setenv("AIGEN_ADMIN_PASSWORD", "custompass")
			},
			cleanupEnv: func() {
				os.Unsetenv("AIGEN_ADMIN_EMAIL")
				os.Unsetenv("AIGEN_ADMIN_PASSWORD")
			},
			expectedEmail: "custom@aigen.local",
			expectedRoles: []string{descriptors.RoleSa, descriptors.RoleAdmin, descriptors.RoleUser},
			expectCreated: true,
		},
		{
			name:         "Skip if admin already exists",
			defaultEmail: "admin@aigen.local",
			defaultPass:  "adminpassword",
			isTestEnv:    true,
			setupEnv:     func() {},
			cleanupEnv:   func() {},
			preExistingUser: &descriptors.User{
				Id:    12345,
				Email: "existing@aigen.local",
				Roles: []string{descriptors.RoleAdmin},
			},
			expectCreated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dao, _ := relationdbdao.CreateDao("memory://")
			dao.EnsureTable(ctx)

			authSvc := NewAuthService(dao, "secret", nil, nil)

			// Setup environment variables if any
			tt.setupEnv()
			defer tt.cleanupEnv()

			// Pre-existing user if specified
			if tt.preExistingUser != nil {
				rec := datamodels.RecJSON{
					Namespace: UserNamespace,
					Key:       tt.preExistingUser.Email,
					Rec:       tt.preExistingUser,
				}
				dao.Save(ctx, rec)
			}

			// Run BootstrapAdmin
			err := authSvc.BootstrapAdmin(ctx, tt.defaultEmail, tt.defaultPass, tt.isTestEnv)
			assert.NoError(t, err)

			if tt.expectCreated {
				// Verify created user exists
				rec, err := dao.Get(ctx, UserNamespace, tt.expectedEmail)
				assert.NoError(t, err)
				assert.NotNil(t, rec)

				// Authenticate
				token, err := authSvc.Login(ctx, tt.expectedEmail, func() string {
					if tt.defaultPass != "" && os.Getenv("AIGEN_ADMIN_PASSWORD") == "" {
						return tt.defaultPass
					}
					if envPass := os.Getenv("AIGEN_ADMIN_PASSWORD"); envPass != "" {
						return envPass
					}
					return "adminpassword" // Fallback default
				}())
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				userId, roles, err := authSvc.ValidateToken(token)
				assert.NoError(t, err)
				assert.True(t, userId > 0)
				assert.Subset(t, roles, tt.expectedRoles)
			} else if tt.preExistingUser != nil {
				// Verify pre-existing remains the only admin
				rec, err := dao.Get(ctx, UserNamespace, tt.preExistingUser.Email)
				assert.NoError(t, err)
				assert.NotNil(t, rec)

				// Verify default was not created
				recDefault, _ := dao.Get(ctx, UserNamespace, "admin@aigen.local")
				assert.Nil(t, recDefault)
			}
		})
	}
}
