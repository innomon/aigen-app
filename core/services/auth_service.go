package services

import (
	"context"
	"fmt"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"encoding/json"
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
		Id:           time.Now().UnixNano(),
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
	data, _ := json.Marshal(rec.Rec)
	json.Unmarshal(data, &user)

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
	data, _ := json.Marshal(recs[0].Rec)
	json.Unmarshal(data, &user)
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
	rec, err := s.dao.Get(ctx, RoleNamespace, name)
	if err != nil || rec == nil {
		return nil, fmt.Errorf("role not found")
	}

	var role descriptors.Role
	data, _ := json.Marshal(rec.Rec)
	json.Unmarshal(data, &role)
	return &role, nil
}

func (s *AuthService) LoginByChannel(ctx context.Context, channelType descriptors.ChannelType, identifier string, token string, ip, ua string) (string, error) {
	// Simplified for the pivot
	return "", fmt.Errorf("LoginByChannel not yet implemented in JSON store model")
}
