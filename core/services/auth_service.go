package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/bcrypt"
)

const (
	UserNamespace     = "aigen.core.descriptors.User"
	RoleNamespace     = "aigen.core.descriptors.Role"
	UserRoleNamespace = "aigen.core.descriptors.UserRole"
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

func (s *AuthService) Register(ctx context.Context, email, password string) (*descriptors.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &descriptors.User{
		Id:           time.Now().Unix(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Roles:        []string{descriptors.RoleUser},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	rec := datamodels.RecJSON{
		Namespace: UserNamespace,
		Key:       email,
		Rec:       user,
		Tmstamp:   time.Now(),
	}

	if err := s.dao.Save(ctx, rec); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	rec, err := s.dao.Get(ctx, UserNamespace, email)
	if err != nil || rec == nil {
		return "", fmt.Errorf("user not found")
	}

	var user descriptors.User
	if err := mapstructure.Decode(rec.Rec, &user); err != nil {
		return "", fmt.Errorf("failed to decode user: %v", err)
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
	if err != nil || len(recs) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	var user descriptors.User
	if err := mapstructure.Decode(recs[0].Rec, &user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %v", err)
	}
	return &user, nil
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
		user = &descriptors.User{
			Id:        now.Unix(),
			Email:     fmt.Sprintf("user_%d@aigen.local", now.Unix()), // Placeholder
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
		s.channelService.RegisterChannel(ctx, user.Id, channelType, identifier, nil)
		s.channelService.VerifyChannel(ctx, user.Id, channelType, "")
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
		s.UpdateUser(ctx, user)
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
	return s.dao.Save(ctx, rec)
}

func (s *AuthService) BootstrapAdmin(ctx context.Context, defaultEmail, defaultPassword string, isTestEnv bool) error {
	// 1. Scan for existing admin users in a database-agnostic way
	recs, _, err := s.dao.List(ctx, UserNamespace, nil, datamodels.Pagination{Limit: func() *string { limitStr := "100"; return &limitStr }()}, nil)
	if err != nil {
		return fmt.Errorf("failed to scan for existing users: %w", err)
	}

	adminExists := false
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

	// If admin already exists, skip bootstrapping
	if adminExists {
		return nil
	}

	// 2. Select credentials to use
	email := defaultEmail
	password := defaultPassword

	// Environment variable overrides take precedence
	if envEmail := os.Getenv("AIGEN_ADMIN_EMAIL"); envEmail != "" {
		email = envEmail
	}
	if envPass := os.Getenv("AIGEN_ADMIN_PASSWORD"); envPass != "" {
		password = envPass
	}

	// Default fallback email
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

	// 3. Register the super-admin user
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

	// 4. Log or display output
	if isRandom {
		printAdminBanner(email, password)
	} else {
		log.Printf("[INFO] Bootstrapped administrator user account: %s", email)
	}

	return nil
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

func printAdminBanner(email, password string) {
	fmt.Println("========================================================================")
	fmt.Println("[WARNING] FIRST TIME STARTUP DETECTED: NO ADMIN ACCOUNT FOUND.")
	fmt.Println("We have safely bootstrapped a super-admin account for you:")
	fmt.Println("")
	fmt.Printf("  Username/Email: %s\n", email)
	fmt.Printf("  Password:       %s\n", password)
	fmt.Println("")
	fmt.Println("Please log in and CHANGE this password immediately inside the admin panel!")
	fmt.Println("========================================================================")
}
