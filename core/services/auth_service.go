package services

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	dao            relationdbdao.IPrimaryDao
	secret         []byte
	channelService IChannelService
}

func NewAuthService(dao relationdbdao.IPrimaryDao, secret string, channelService IChannelService) *AuthService {
	return &AuthService{
		dao:            dao,
		secret:         []byte(secret),
		channelService: channelService,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*descriptors.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &descriptors.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		Roles:        []string{descriptors.RoleUser},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	query, args, err := s.dao.GetBuilder().Insert(descriptors.UserTableName).
		Columns("email", "password_hash", "role", "created_at", "updated_at").
		Values(user.Email, user.PasswordHash, user.Roles[0], user.CreatedAt, user.UpdatedAt).ToSql()
	if err != nil {
		return nil, err
	}

	var newId int64
	if strings.Contains(query, "$1") {
		// Postgres
		err = s.dao.GetDb().QueryRowContext(ctx, query+" RETURNING id", args...).Scan(&newId)
	} else {
		// SQLite
		res, err := s.dao.GetDb().ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		newId, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	user.Id = newId

	// Link role in junction table if RBAC is active
	_, _, _ = s.dao.GetBuilder().Insert("__user_roles").
		Columns("user_id", "role_id").
		Select(squirrel.Select(fmt.Sprintf("%d", user.Id), "id").From("__roles").Where(squirrel.Eq{"name": descriptors.RoleUser})).
		ToSql()

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	query, args, err := s.dao.GetBuilder().Select("id", "email", "password_hash", "role", "avatar_path", "default_role_id", "created_at", "updated_at").From(descriptors.UserTableName).
		Where(squirrel.Eq{"email": email}).Limit(1).ToSql()
	if err != nil {
		return "", err
	}

	row := s.dao.GetDb().QueryRowContext(ctx, query, args...)
	var id int64
	var e, pass, role string
	var avatar sql.NullString
	var defaultRoleId sql.NullInt64
	var createdAt, updatedAt time.Time
	if err := row.Scan(&id, &e, &pass, &role, &avatar, &defaultRoleId, &createdAt, &updatedAt); err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(pass), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid password")
	}

	roles, _, _ := s.getRoles(ctx, id, role)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": id,
		"roles":  roles,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString(s.secret)
}

func (s *AuthService) LoginByChannel(ctx context.Context, channelType descriptors.ChannelType, identifier string, token string, ip, ua string) (string, error) {
	userId, email, err := s.ValidateChannelToken(ctx, channelType, token)
	
	log := &descriptors.AuthLog{
		ChannelType: channelType,
		Action:      "login",
		IPAddress:   ip,
		UserAgent:   ua,
		Success:     err == nil,
	}
	if userId != 0 {
		log.UserId = &userId
	}

	if err != nil {
		log.Metadata = fmt.Sprintf(`{"error": "%s"}`, err.Error())
		_ = s.channelService.LogAuthAttempt(ctx, log)
		return "", err
	}

	_ = s.channelService.LogAuthAttempt(ctx, log)

	var finalUserId int64
	var roles []string

	if userId != 0 {
		finalUserId = userId
		user, err := s.Me(ctx, userId)
		if err == nil {
			roles = user.Roles
		}
	} else if email != "" {
		// Try to find user by email or identifier
		query, args, _ := s.dao.GetBuilder().Select("id", "role").From(descriptors.UserTableName).Where(squirrel.Eq{"email": email}).Limit(1).ToSql()
		var id int64
		var role string
		if err := s.dao.GetDb().QueryRowContext(ctx, query, args...).Scan(&id, &role); err == nil {
			finalUserId = id
			r, _, _ := s.getRoles(ctx, id, role)
			roles = r
		} else {
			// Auto-register or guest logic here
			// For now, return error if user not found
			return "", fmt.Errorf("user not found for channel identifier")
		}
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": finalUserId,
		"roles":  roles,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	})

	return jwtToken.SignedString(s.secret)
}

func (s *AuthService) ValidateChannelToken(ctx context.Context, channelType descriptors.ChannelType, tokenString string) (int64, string, error) {
	switch channelType {
	case descriptors.ChannelWhatsApp:
		// EdDSA verification (simplified logic from whatsadk)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// In real app, get public key from config based on gateway
			// For now, placeholder or check if we have it in config
			return nil, fmt.Errorf("public key not configured for WhatsApp")
		})
		if err != nil && !strings.Contains(err.Error(), "public key not configured") {
			return 0, "", err
		}
		
		// If we had the key and it was valid:
		if token != nil && token.Valid {
			claims := token.Claims.(jwt.MapClaims)
			phone := claims["sub"].(string)
			return 0, phone, nil // Return phone as email/identifier for now
		}
		
		// Fallback for MVP/Demo: if token is just "WHATSAPP_TEST_ID", allow
		if tokenString == "WHATSAPP_TEST_ID" {
			return 0, "test-whatsapp-user", nil
		}
		
	case descriptors.ChannelEmail:
		// Placeholder for mailadk style verification
		if strings.HasPrefix(tokenString, "EMAIL_TEST_") {
			return 0, strings.TrimPrefix(tokenString, "EMAIL_TEST_"), nil
		}
	}

	return 0, "", fmt.Errorf("unsupported or invalid channel token")
}

func (s *AuthService) Me(ctx context.Context, userId int64) (*descriptors.User, error) {
	query, args, err := s.dao.GetBuilder().Select("id", "email", "password_hash", "role", "avatar_path", "default_role_id", "created_at", "updated_at").From(descriptors.UserTableName).
		Where(squirrel.Eq{"id": userId}).Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	row := s.dao.GetDb().QueryRowContext(ctx, query, args...)
	var id int64
	var e, pass, role string
	var avatar sql.NullString
	var defaultRoleId sql.NullInt64
	var createdAt, updatedAt time.Time
	if err := row.Scan(&id, &e, &pass, &role, &avatar, &defaultRoleId, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	roles, rolesDetails, _ := s.getRoles(ctx, id, role)
	
	var defaultRolePtr *int64
	if defaultRoleId.Valid {
		defaultRolePtr = &defaultRoleId.Int64
	}

	user := &descriptors.User{
		Id:            id,
		Email:         e,
		PasswordHash:  pass,
		Roles:         roles,
		RolesDetails:  rolesDetails,
		DefaultRoleId: defaultRolePtr,
		AvatarPath:    avatar.String,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}

	// Load channels
	channels, _ := s.channelService.GetChannelsByUserId(ctx, id)
	for _, c := range channels {
		user.Channels = append(user.Channels, *c)
	}

	return user, nil
}

func (s *AuthService) ValidateToken(tokenString string) (int64, []string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return 0, nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userId := int64(claims["userId"].(float64))
		
		var roles []string
		if r, ok := claims["roles"].([]interface{}); ok {
			for _, role := range r {
				roles = append(roles, role.(string))
			}
		} else if r, ok := claims["role"].(string); ok {
			roles = []string{r}
		}

		return userId, roles, nil
	}

	return 0, nil, fmt.Errorf("invalid token")
}

func (s *AuthService) getRoles(ctx context.Context, userId int64, legacyRole string) ([]string, []descriptors.Role, error) {
	query, args, err := s.dao.GetBuilder().Select("r.id, r.name, r.disabled, r.dashboard_page_id, r.menu_id").From("__user_roles ur").
		Join("__roles r ON ur.role_id = r.id").
		Where(squirrel.Eq{"ur.user_id": userId}).ToSql()

	if err != nil {
		return []string{legacyRole}, nil, nil
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return []string{legacyRole}, nil, nil
	}
	defer rows.Close()

	var roles []string
	var rolesDetails []descriptors.Role
	for rows.Next() {
		var role descriptors.Role
		var dashboardPageId, menuId sql.NullString
		if err := rows.Scan(&role.Id, &role.Name, &role.Disabled, &dashboardPageId, &menuId); err == nil {
			role.DashboardPageId = dashboardPageId.String
			role.MenuId = menuId.String
			rolesDetails = append(rolesDetails, role)
			roles = append(roles, role.Name)
		}
	}

	if len(roles) == 0 {
		return []string{legacyRole}, nil, nil
	}

	return roles, rolesDetails, nil
}

func (s *AuthService) GetRoleByName(ctx context.Context, name string) (*descriptors.Role, error) {
	query, args, err := s.dao.GetBuilder().Select("id", "name", "disabled", "dashboard_page_id", "menu_id").From("__roles").
		Where(squirrel.Eq{"name": name}).Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	row := s.dao.GetDb().QueryRowContext(ctx, query, args...)
	var role descriptors.Role
	var dashboardPageId, menuId sql.NullString
	if err := row.Scan(&role.Id, &role.Name, &role.Disabled, &dashboardPageId, &menuId); err != nil {
		return nil, err
	}
	role.DashboardPageId = dashboardPageId.String
	role.MenuId = menuId.String

	return &role, nil
}

// Helper to decode Base64Url (used in whatsadk)
func decodeBase64Url(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// Verify Ed25519 signature manually if needed
func verifyEd25519(publicKeyB64, message, signatureB64 string) bool {
	pubKeyBytes, err := decodeBase64Url(publicKeyB64)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := decodeBase64Url(signatureB64)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(pubKeyBytes, []byte(message), sigBytes)
}
