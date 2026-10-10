package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/innomon/aigen-app/utils/logger"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/innomon/aigen-app/utils/ids"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/bcrypt"
)

const (
	UserNamespace       = "aigen.core.descriptors.User"
	EntityUserNamespace = "aigen.bizdef.entities.User"
	RoleNamespace       = "aigen.core.descriptors.Role"
	UserRoleNamespace   = "aigen.core.descriptors.UserRole"
)

type AuthService struct {
	dao            relationdbdao.IPrimaryDao
	secret         []byte
	channelService IChannelService
	whatsappService IWhatsAppService
}

func NewAuthService(dao relationdbdao.IPrimaryDao, secret string, channelService IChannelService, whatsappService IWhatsAppService) *AuthService {
	return &AuthService{
		dao:             dao,
		secret:          []byte(secret),
		channelService:  channelService,
		whatsappService: whatsappService,
	}
}

func (s *AuthService) convertEntityMapToUser(data map[string]interface{}, key string) *descriptors.User {
	user := &descriptors.User{}

	if email, ok := data["email"].(string); ok {
		user.Email = email
	}
	if phone, ok := data["phone"].(string); ok {
		user.Phone = phone
	}
	if pwHash, ok := data["password_hash"].(string); ok {
		user.PasswordHash = pwHash
	}
	if avatar, ok := data["avatar_path"].(string); ok {
		user.AvatarPath = avatar
	}

	rawID := data["id"]
	if rawID == nil || rawID == "" {
		rawID = key
	}
	switch v := rawID.(type) {
	case int64:
		user.Id = v
	case float64:
		user.Id = int64(v)
	case int:
		user.Id = int64(v)
	default:
		user.Id = ids.StringToInt64ID(fmt.Sprintf("%v", v))
	}

	var roles []string
	if rolesRaw, ok := data["roles"].([]interface{}); ok && len(rolesRaw) > 0 {
		for _, r := range rolesRaw {
			if rStr, ok := r.(string); ok && rStr != "" {
				roles = append(roles, rStr)
			}
		}
	} else if rolesStrList, ok := data["roles"].([]string); ok && len(rolesStrList) > 0 {
		roles = rolesStrList
	}

	if len(roles) == 0 {
		roleStr, _ := data["role"].(string)
		switch roleStr {
		case descriptors.RoleSa, "superadmin":
			roles = []string{descriptors.RoleSa, descriptors.RoleAdmin, descriptors.RoleUser}
		case descriptors.RoleAdmin:
			roles = []string{descriptors.RoleAdmin, descriptors.RoleUser}
		case descriptors.RoleUser:
			roles = []string{descriptors.RoleUser}
		case descriptors.RoleGuest:
			roles = []string{descriptors.RoleGuest}
		case "":
			roles = []string{descriptors.RoleAdmin, descriptors.RoleUser}
		default:
			roles = []string{roleStr, descriptors.RoleUser}
		}
	}
	user.Roles = roles

	if caStr, ok := data["created_at"].(string); ok && caStr != "" {
		if t, err := time.Parse(time.RFC3339Nano, caStr); err == nil {
			user.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339, caStr); err == nil {
			user.CreatedAt = t
		}
	} else if caTime, ok := data["created_at"].(time.Time); ok {
		user.CreatedAt = caTime
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}

	if uaStr, ok := data["updated_at"].(string); ok && uaStr != "" {
		if t, err := time.Parse(time.RFC3339Nano, uaStr); err == nil {
			user.UpdatedAt = t
		} else if t, err := time.Parse(time.RFC3339, uaStr); err == nil {
			user.UpdatedAt = t
		}
	} else if uaTime, ok := data["updated_at"].(time.Time); ok {
		user.UpdatedAt = uaTime
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now()
	}

	return user
}

func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*descriptors.User, error) {
	// 1. Try UserNamespace
	rec, err := s.dao.Get(ctx, UserNamespace, email)
	if err == nil && rec != nil {
		var user descriptors.User
		if err := mapstructure.Decode(rec.Rec, &user); err == nil && user.Email != "" {
			return &user, nil
		}
	}

	// 2. Try EntityUserNamespace
	filters := []datamodels.Filter{
		{
			FieldName: "email",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{email}},
			},
		},
	}
	limStr := "1"
	recs, _, err := s.dao.List(ctx, EntityUserNamespace, filters, datamodels.Pagination{Limit: &limStr}, nil)
	if err != nil || len(recs) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	entityRec := recs[0].Rec
	entityMap, ok := entityRec.(map[string]interface{})
	if !ok {
		if err := mapstructure.Decode(entityRec, &entityMap); err != nil {
			return nil, fmt.Errorf("failed to parse entity user: %w", err)
		}
	}

	user := s.convertEntityMapToUser(entityMap, recs[0].Key)

	// Backfill into UserNamespace
	_ = s.dao.Save(ctx, datamodels.RecJSON{
		Namespace: UserNamespace,
		Key:       user.Email,
		Rec:       user,
		Tmstamp:   time.Now(),
	})

	return user, nil
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*descriptors.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &descriptors.User{
		Id:           ids.NewRandomInt64ID(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Roles:        []string{descriptors.RoleUser},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	rec := datamodels.RecJSON{
		Namespace: UserNamespace,
		Key:       email,
		Rec:       user,
		Tmstamp:   now,
	}

	if err := s.dao.Save(ctx, rec); err != nil {
		return nil, err
	}

	entityRec := datamodels.RecJSON{
		Namespace: EntityUserNamespace,
		Key:       fmt.Sprintf("%d", user.Id),
		Rec: map[string]interface{}{
			"id":            fmt.Sprintf("%d", user.Id),
			"email":         user.Email,
			"password_hash": user.PasswordHash,
			"role":          descriptors.RoleUser,
			"roles":         user.Roles,
			"created_at":    now,
			"updated_at":    now,
		},
		Tmstamp: now,
	}
	_ = s.dao.Save(ctx, entityRec)

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return "", fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": user.Id,
		"email":  user.Email,
		"roles":  user.Roles,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString(s.secret)
}

func (s *AuthService) Me(ctx context.Context, userId int64) (*descriptors.User, error) {
	// Need to find user by ID. Key is email.
	filters := []datamodels.Filter{
		{FieldName: "id", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{userId}}}},
	}
	recs, _, err := s.dao.List(ctx, UserNamespace, filters, datamodels.Pagination{}, nil)
	var user *descriptors.User
	if err == nil && len(recs) > 0 {
		var u descriptors.User
		if err := mapstructure.Decode(recs[0].Rec, &u); err == nil {
			user = &u
		}
	}

	if user == nil {
		allRecs, _, err := s.dao.List(ctx, EntityUserNamespace, nil, datamodels.Pagination{}, nil)
		if err == nil {
			for _, r := range allRecs {
				if rMap, ok := r.Rec.(map[string]interface{}); ok {
					u := s.convertEntityMapToUser(rMap, r.Key)
					if u.Id == userId {
						user = u
						_ = s.dao.Save(ctx, datamodels.RecJSON{
							Namespace: UserNamespace,
							Key:       user.Email,
							Rec:       user,
							Tmstamp:   time.Now(),
						})
						break
					}
				}
			}
		}
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	if len(user.RolesDetails) == 0 && len(user.Roles) > 0 {
		for i, r := range user.Roles {
			roleObj, err := s.GetRoleByName(ctx, r)
			if err == nil && roleObj != nil {
				user.RolesDetails = append(user.RolesDetails, *roleObj)
			} else {
				user.RolesDetails = append(user.RolesDetails, descriptors.Role{
					Id:   int64(i + 1),
					Name: r,
				})
			}
		}
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
		var userId int64
		switch v := claims["userId"].(type) {
		case float64:
			userId = int64(v)
		case int64:
			userId = v
		case int:
			userId = int64(v)
		default:
			return 0, nil, fmt.Errorf("invalid userId claim type")
		}
		
		var roles []string
		if r, ok := claims["roles"].([]interface{}); ok {
			for _, role := range r {
				if rStr, ok := role.(string); ok {
					roles = append(roles, rStr)
				}
			}
		}
		return userId, roles, nil
	}

	return 0, nil, fmt.Errorf("invalid token")
}

func (s *AuthService) GetRoleByName(ctx context.Context, name string) (*descriptors.Role, error) {
	filters := []datamodels.Filter{
		{
			FieldName: "name",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{name}},
			},
		},
	}
	
	limitStr := "1"
	recs, _, err := s.dao.List(ctx, "aigen.bizdef.entities.Role", filters, datamodels.Pagination{Limit: &limitStr}, nil)
	if err != nil || len(recs) == 0 {
		return nil, fmt.Errorf("role not found")
	}

	var role descriptors.Role
	if err := mapstructure.Decode(recs[0].Rec, &role); err != nil {
		return nil, fmt.Errorf("failed to decode role: %v", err)
	}
	return &role, nil
}


func (s *AuthService) LoginByChannel(ctx context.Context, channelType descriptors.ChannelType, identifier string, token string, ip, ua string) (string, error) {
	// 1. Check if channel already linked to a user
	userChannel, err := s.channelService.GetChannelByIdentifier(ctx, channelType, identifier)
	var user *descriptors.User
	
	if err == nil && userChannel != nil {
		user, err = s.Me(ctx, userChannel.UserId)
		if err != nil {
			return "", fmt.Errorf("user linked to channel not found: %w", err)
		}
	} else {
		// 2. Not linked. Check if a user exists with this identifier as their "Phone" (if WhatsApp)
		if channelType == descriptors.ChannelWhatsApp {
			filters := []datamodels.Filter{
				{FieldName: "phone", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{identifier}}}},
			}
			recs, _, err := s.dao.List(ctx, UserNamespace, filters, datamodels.Pagination{Limit: func() *string { s := "1"; return &s }()}, nil)
			if err == nil && len(recs) > 0 {
				if err := mapstructure.Decode(recs[0].Rec, &user); err == nil {
					// Link the channel
					s.channelService.RegisterChannel(ctx, user.Id, channelType, identifier, nil)
					s.channelService.VerifyChannel(ctx, user.Id, channelType, "")
				}
			}
		}
	}

	// 3. If still no user, create a new one (only if not doing TOTP verification with missing user)
	if user == nil {
		if token != "" {
			return "", fmt.Errorf("user not found for TOTP verification")
		}

		now := time.Now()
		newId := ids.NewRandomInt64ID()
		user = &descriptors.User{
			Id:        newId,
			Email:     fmt.Sprintf("user_%d@aigen.local", newId), // Placeholder
			Phone:     identifier,
			Roles:     []string{descriptors.RoleUser},
			CreatedAt: now,
			UpdatedAt: now,
		}
		
		rec := datamodels.RecJSON{
			Namespace: UserNamespace,
			Key:       user.Email,
			Rec:       user,
			Tmstamp:   now,
		}
		if err := s.dao.Save(ctx, rec); err != nil {
			return "", fmt.Errorf("failed to create new user: %w", err)
		}

		// Link the channel
		_, _ = s.channelService.RegisterChannel(ctx, user.Id, channelType, identifier, nil)
		_, _ = s.channelService.VerifyChannel(ctx, user.Id, channelType, "")
	}

	// 4. TOTP Verification if token is provided
	if token != "" && channelType == descriptors.ChannelWhatsApp {
		if len(user.TOTPSecret) > 0 {
			var code uint32
			fmt.Sscanf(token, "%d", &code)
			
			appID := "AIGenApp"
			if !s.whatsappService.VerifyTOTPCode(user.TOTPSecret, user.TOTPPubKey, fmt.Sprintf("%d", user.Id), appID, code) {
				return "", fmt.Errorf("invalid TOTP code")
			}
		} else {
			// If no TOTP secret, but token provided, maybe it's some other verification?
			// For now, we only support TOTP if secret is set.
			return "", fmt.Errorf("TOTP not enrolled for this user")
		}
	}

	// 5. Issue Token
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": user.Id,
		"email":  user.Email,
		"roles":  user.Roles,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenStr, err := jwtToken.SignedString(s.secret)
	if err != nil {
		return "", err
	}

	// 6. Log attempt
	s.channelService.LogAuthAttempt(ctx, &descriptors.AuthLog{
		UserId:      &user.Id,
		ChannelType: channelType,
		Action:      "login",
		IPAddress:   ip,
		UserAgent:   ua,
		Success:     true,
	})

	return tokenStr, nil
}


func (s *AuthService) LinkChannel(ctx context.Context, userId int64, channelType descriptors.ChannelType, identifier string) error {
	user, err := s.Me(ctx, userId)
	if err != nil {
		return err
	}

	// Update user phone if it's WhatsApp and not set
	if channelType == descriptors.ChannelWhatsApp && user.Phone == "" {
		user.Phone = identifier
		if err := s.UpdateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to update user phone: %w", err)
		}
	}

	_, err = s.channelService.RegisterChannel(ctx, userId, channelType, identifier, nil)
	if err != nil {
		return err
	}
	
	_, err = s.channelService.VerifyChannel(ctx, userId, channelType, "")
	return err
}

func (s *AuthService) UpdateUser(ctx context.Context, user *descriptors.User) error {
	user.UpdatedAt = time.Now()
	rec := datamodels.RecJSON{
		Namespace: UserNamespace,
		Key:       user.Email,
		Rec:       user,
		Tmstamp:   user.UpdatedAt,
	}
	if err := s.dao.Save(ctx, rec); err != nil {
		return err
	}

	entityKey := fmt.Sprintf("%d", user.Id)
	roleVal := descriptors.RoleUser
	if len(user.Roles) > 0 {
		roleVal = user.Roles[0]
	}
	entityRec := datamodels.RecJSON{
		Namespace: EntityUserNamespace,
		Key:       entityKey,
		Rec: map[string]interface{}{
			"id":            entityKey,
			"email":         user.Email,
			"phone":         user.Phone,
			"avatar_path":   user.AvatarPath,
			"password_hash": user.PasswordHash,
			"role":          roleVal,
			"roles":         user.Roles,
			"created_at":    user.CreatedAt,
			"updated_at":    user.UpdatedAt,
		},
		Tmstamp: user.UpdatedAt,
	}
	_ = s.dao.Save(ctx, entityRec)

	return nil
}

func (s *AuthService) syncUserStores(ctx context.Context) {
	// 1. Sync from EntityUserNamespace to UserNamespace
	entityRecs, _, err := s.dao.List(ctx, EntityUserNamespace, nil, datamodels.Pagination{}, nil)
	if err == nil {
		for _, er := range entityRecs {
			var entityMap map[string]interface{}
			if m, ok := er.Rec.(map[string]interface{}); ok {
				entityMap = m
			} else {
				if err := mapstructure.Decode(er.Rec, &entityMap); err != nil {
					continue
				}
			}
			user := s.convertEntityMapToUser(entityMap, er.Key)
			if user.Email == "" {
				continue
			}

			// Ensure entity record itself has role/roles populated
			updatedEntity := false
			if entityMap["role"] == nil || fmt.Sprintf("%v", entityMap["role"]) == "" {
				entityMap["role"] = descriptors.RoleAdmin
				updatedEntity = true
			}
			if entityMap["roles"] == nil {
				entityMap["roles"] = user.Roles
				updatedEntity = true
			}
			if updatedEntity {
				_ = s.dao.Save(ctx, datamodels.RecJSON{
					Namespace: EntityUserNamespace,
					Key:       er.Key,
					Rec:       entityMap,
					Tmstamp:   time.Now(),
				})
			}

			// Ensure user exists in UserNamespace
			uRec, err := s.dao.Get(ctx, UserNamespace, user.Email)
			if err != nil || uRec == nil {
				_ = s.dao.Save(ctx, datamodels.RecJSON{
					Namespace: UserNamespace,
					Key:       user.Email,
					Rec:       user,
					Tmstamp:   user.UpdatedAt,
				})
			}
		}
	}

	// 2. Sync from UserNamespace to EntityUserNamespace
	userRecs, _, err := s.dao.List(ctx, UserNamespace, nil, datamodels.Pagination{}, nil)
	if err == nil {
		for _, ur := range userRecs {
			var u descriptors.User
			if err := mapstructure.Decode(ur.Rec, &u); err != nil || u.Email == "" {
				continue
			}
			entityKey := fmt.Sprintf("%d", u.Id)
			eRec, err := s.dao.Get(ctx, EntityUserNamespace, entityKey)
			if err != nil || eRec == nil {
				// Also check by email in EntityUserNamespace
				filters := []datamodels.Filter{
					{FieldName: "email", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{u.Email}}}},
				}
				lim := "1"
				matching, _, mErr := s.dao.List(ctx, EntityUserNamespace, filters, datamodels.Pagination{Limit: &lim}, nil)
				if mErr == nil && len(matching) > 0 {
					continue
				}

				roleVal := descriptors.RoleUser
				if len(u.Roles) > 0 {
					roleVal = u.Roles[0]
				}
				_ = s.dao.Save(ctx, datamodels.RecJSON{
					Namespace: EntityUserNamespace,
					Key:       entityKey,
					Rec: map[string]interface{}{
						"id":            entityKey,
						"email":         u.Email,
						"phone":         u.Phone,
						"avatar_path":   u.AvatarPath,
						"password_hash": u.PasswordHash,
						"role":          roleVal,
						"roles":         u.Roles,
						"created_at":    u.CreatedAt,
						"updated_at":    u.UpdatedAt,
					},
					Tmstamp: u.UpdatedAt,
				})
			}
		}
	}
}

func (s *AuthService) BootstrapRoles(ctx context.Context) error {
	defaultRoles := []descriptors.Role{
		{Id: 1, Name: descriptors.RoleSa, Disabled: false},
		{Id: 2, Name: descriptors.RoleAdmin, Disabled: false},
		{Id: 3, Name: descriptors.RoleUser, Disabled: false},
		{Id: 4, Name: descriptors.RoleGuest, Disabled: false},
	}

	for _, r := range defaultRoles {
		filters := []datamodels.Filter{
			{FieldName: "name", Constraints: []datamodels.Constraint{{Match: "equals", Values: []interface{}{r.Name}}}},
		}
		limStr := "1"
		recs, _, err := s.dao.List(ctx, "aigen.bizdef.entities.Role", filters, datamodels.Pagination{Limit: &limStr}, nil)
		if err == nil && len(recs) == 0 {
			now := time.Now()
			rec := datamodels.RecJSON{
				Namespace: "aigen.bizdef.entities.Role",
				Key:       fmt.Sprintf("%d", r.Id),
				Rec: map[string]interface{}{
					"id":        r.Id,
					"name":      r.Name,
					"disabled":  r.Disabled,
					"created_at": now,
					"updated_at": now,
				},
				Tmstamp: now,
			}
			_ = s.dao.Save(ctx, rec)
		}
	}
	return nil
}

func (s *AuthService) BootstrapAdmin(ctx context.Context, defaultEmail, defaultPassword string, isTestEnv bool) error {
	_ = s.BootstrapRoles(ctx)
	s.syncUserStores(ctx)
	defer s.syncUserStores(ctx)

	// 1. Select credentials to use and check for explicit environment variable configuration
	email := defaultEmail
	password := defaultPassword

	hasExplicitEnv := false
	for _, k := range []string{"SA_USER", "SA_EMAIL", "SA_USER_ID", "AIGEN_ADMIN_EMAIL", "ADMIN_EMAIL", "ADMIN_USER"} {
		if envVal := os.Getenv(k); envVal != "" {
			email = envVal
			hasExplicitEnv = true
			break
		}
	}
	for _, k := range []string{"SA_PASSWORD", "SA_PASS", "AIGEN_ADMIN_PASSWORD", "ADMIN_PASSWORD", "ADMIN_PASS"} {
		if envVal := os.Getenv(k); envVal != "" {
			password = envVal
			hasExplicitEnv = true
			break
		}
	}

	// If explicit credentials were provided via environment variables, ensure that account is configured
	if hasExplicitEnv && email != "" {
		if password == "" {
			if isTestEnv {
				password = "adminpassword"
			} else {
				randPass, err := generateSecurePassword(16)
				if err != nil {
					return fmt.Errorf("failed to generate secure random password: %w", err)
				}
				password = randPass
				saveCredentialsToFile(email, password)
			}
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		existingRec, err := s.dao.Get(ctx, UserNamespace, email)
		now := time.Now()
		if err == nil && existingRec != nil {
			// Update existing user credentials and ensure admin/sa roles
			var user descriptors.User
			if err := mapstructure.Decode(existingRec.Rec, &user); err == nil {
				user.PasswordHash = string(hashedPassword)
				user.UpdatedAt = now
				hasSa := false
				hasAdmin := false
				hasUser := false
				for _, r := range user.Roles {
					if r == descriptors.RoleSa {
						hasSa = true
					}
					if r == descriptors.RoleAdmin {
						hasAdmin = true
					}
					if r == descriptors.RoleUser {
						hasUser = true
					}
				}
				if !hasSa {
					user.Roles = append(user.Roles, descriptors.RoleSa)
				}
				if !hasAdmin {
					user.Roles = append(user.Roles, descriptors.RoleAdmin)
				}
				if !hasUser {
					user.Roles = append(user.Roles, descriptors.RoleUser)
				}
				rec := datamodels.RecJSON{
					Namespace: UserNamespace,
					Key:       email,
					Rec:       user,
					Tmstamp:   now,
				}
				if err := s.dao.Save(ctx, rec); err != nil {
					return fmt.Errorf("failed to update admin user: %w", err)
				}
				entityKey := fmt.Sprintf("%d", user.Id)
				_ = s.dao.Save(ctx, datamodels.RecJSON{
					Namespace: EntityUserNamespace,
					Key:       entityKey,
					Rec: map[string]interface{}{
						"id":            entityKey,
						"email":         user.Email,
						"password_hash": user.PasswordHash,
						"role":          descriptors.RoleSa,
						"roles":         user.Roles,
						"created_at":    user.CreatedAt,
						"updated_at":    now,
					},
					Tmstamp: now,
				})
				logger.Printf("[INFO] Synchronized super-admin credentials for: %s", email)
			}
		} else {
			// Create new super-admin user
			user := &descriptors.User{
				Id:           now.Unix(),
				Email:        email,
				PasswordHash: string(hashedPassword),
				Roles:        []string{descriptors.RoleSa, descriptors.RoleAdmin, descriptors.RoleUser},
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			rec := datamodels.RecJSON{
				Namespace: UserNamespace,
				Key:       email,
				Rec:       user,
				Tmstamp:   now,
			}
			if err := s.dao.Save(ctx, rec); err != nil {
				return fmt.Errorf("failed to save bootstrapped admin user: %w", err)
			}
			entityKey := fmt.Sprintf("%d", user.Id)
			_ = s.dao.Save(ctx, datamodels.RecJSON{
				Namespace: EntityUserNamespace,
				Key:       entityKey,
				Rec: map[string]interface{}{
					"id":            entityKey,
					"email":         user.Email,
					"password_hash": user.PasswordHash,
					"role":          descriptors.RoleSa,
					"roles":         user.Roles,
					"created_at":    now,
					"updated_at":    now,
				},
				Tmstamp: now,
			})
			logger.Printf("[INFO] Bootstrapped administrator user account: %s", email)
		}

		// If configured with a custom SA account, clean up default demo accounts to avoid public exposure
		for _, legacyDefault := range []string{"admin@erp.local", "admin@aigen.local"} {
			if email != legacyDefault {
				if defRec, _ := s.dao.Get(ctx, UserNamespace, legacyDefault); defRec != nil {
					var defUser descriptors.User
					if mapstructure.Decode(defRec.Rec, &defUser) == nil {
						// Only remove if it has the default legacy password
						if bcrypt.CompareHashAndPassword([]byte(defUser.PasswordHash), []byte("admin")) == nil ||
							bcrypt.CompareHashAndPassword([]byte(defUser.PasswordHash), []byte("adminpassword")) == nil {
							_ = s.dao.Delete(ctx, UserNamespace, legacyDefault)
							_ = s.dao.Delete(ctx, EntityUserNamespace, fmt.Sprintf("%d", defUser.Id))
							logger.Printf("[INFO] Removed legacy default account with default password: %s", legacyDefault)
						}
					}
				}
			}
		}
		return nil
	}

	// 2. Scan for existing admin users in a database-agnostic way (paginating all records)
	adminExists := false
	offset := uint64(0)
	limit := uint64(100)

	for {
		limStr := fmt.Sprintf("%d", limit)
		offStr := fmt.Sprintf("%d", offset)
		recs, total, err := s.dao.List(ctx, UserNamespace, nil, datamodels.Pagination{
			Limit:  &limStr,
			Offset: &offStr,
		}, nil)
		if err != nil {
			return fmt.Errorf("failed to scan for existing users: %w", err)
		}

		for _, rec := range recs {
			var user descriptors.User
			if err := mapstructure.Decode(rec.Rec, &user); err == nil {
				for _, role := range user.Roles {
					if role == descriptors.RoleAdmin || role == descriptors.RoleSa {
						adminExists = true
						break
					}
				}
			}
			if adminExists {
				break
			}
		}

		if adminExists || len(recs) == 0 || offset+uint64(len(recs)) >= uint64(total) {
			break
		}
		offset += limit
	}

	// If admin already exists, skip bootstrapping
	if adminExists {
		return nil
	}

	// 3. Fallback credentials when no admin exists and no env provided
	if email == "" {
		email = "admin@aigen.local"
	}

	isRandom := false
	if password == "" {
		if isTestEnv {
			password = "adminpassword"
		} else {
			randPass, err := generateSecurePassword(16)
			if err != nil {
				return fmt.Errorf("failed to generate secure random password: %w", err)
			}
			password = randPass
			isRandom = true
		}
	}

	// 4. Register the super-admin user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &descriptors.User{
		Id:           now.Unix(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Roles:        []string{descriptors.RoleSa, descriptors.RoleAdmin, descriptors.RoleUser},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	rec := datamodels.RecJSON{
		Namespace: UserNamespace,
		Key:       email,
		Rec:       user,
		Tmstamp:   now,
	}

	if err := s.dao.Save(ctx, rec); err != nil {
		return fmt.Errorf("failed to save bootstrapped admin user: %w", err)
	}

	entityKey := fmt.Sprintf("%d", user.Id)
	_ = s.dao.Save(ctx, datamodels.RecJSON{
		Namespace: EntityUserNamespace,
		Key:       entityKey,
		Rec: map[string]interface{}{
			"id":            entityKey,
			"email":         user.Email,
			"password_hash": user.PasswordHash,
			"role":          descriptors.RoleSa,
			"roles":         user.Roles,
			"created_at":    now,
			"updated_at":    now,
		},
		Tmstamp: now,
	})

	// 5. Securely persist or log credentials without displaying passwords on screen
	if isRandom {
		saveCredentialsToFile(email, password)
		printAdminBanner(email)
	} else {
		logger.Printf("[INFO] Bootstrapped administrator user account: %s", email)
	}

	return nil
}

func saveCredentialsToFile(email, password string) {
	_ = os.MkdirAll("logs", 0750)
	content := fmt.Sprintf("# Super-Admin Initial Credentials\n# Generated at: %s\n# DO NOT COMMIT THIS FILE\nSA_USER=%s\nSA_PASSWORD=%s\n", time.Now().Format(time.RFC3339), email, password)
	_ = os.WriteFile("logs/.admin_credentials", []byte(content), 0600)
}

func generateSecurePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}
	return string(result), nil
}

func printAdminBanner(email string) {
	fmt.Println("========================================================================")
	fmt.Println("[SECURITY] FIRST TIME STARTUP DETECTED: NO ADMIN ACCOUNT FOUND.")
	fmt.Println("A super-admin account has been securely initialized.")
	fmt.Printf("  Username/Email: %s\n", email)
	fmt.Println("  Password:       [PROTECTED - Never displayed on screen]")
	fmt.Println("Credentials have been saved to: logs/.admin_credentials (mode 0600)")
	fmt.Println("To set custom credentials, configure SA_USER and SA_PASSWORD in .env")
	fmt.Println("========================================================================")
}
