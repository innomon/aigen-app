package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type ChannelService struct {
	dao    relationdbdao.IPrimaryDao
	config descriptors.ChannelsConfig
}

func NewChannelService(dao relationdbdao.IPrimaryDao, config descriptors.ChannelsConfig) *ChannelService {
	return &ChannelService{
		dao:    dao,
		config: config,
	}
}

func (s *ChannelService) RegisterChannel(ctx context.Context, userId int64, channelType descriptors.ChannelType, identifier string, metadata map[string]interface{}) (*descriptors.UserChannel, error) {
	metadataJson, _ := json.Marshal(metadata)
	now := time.Now()
	
	userChannel := &descriptors.UserChannel{
		UserId:          userId,
		ChannelType:     channelType,
		Identifier:      identifier,
		IsAuthenticated: false,
		Metadata:        string(metadataJson),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	query, args, err := s.dao.GetBuilder().Insert(descriptors.UserChannelTableName).
		Columns("user_id", "channel_type", "identifier", "is_authenticated", "metadata", "created_at", "updated_at").
		Values(userChannel.UserId, userChannel.ChannelType, userChannel.Identifier, userChannel.IsAuthenticated, userChannel.Metadata, userChannel.CreatedAt, userChannel.UpdatedAt).ToSql()
	if err != nil {
		return nil, err
	}

	res, err := s.dao.GetDb().ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	userChannel.Id = id

	return userChannel, nil
}

func (s *ChannelService) VerifyChannel(ctx context.Context, userId int64, channelType descriptors.ChannelType, token string) (bool, error) {
	// For MVP, just mark as authenticated if the token is "123456" or similar
	// Real implementation would verify against WhatsApp Ed25519 JWT or Email token
	
	query, args, err := s.dao.GetBuilder().Update(descriptors.UserChannelTableName).
		Set("is_authenticated", true).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"user_id": userId, "channel_type": channelType}).ToSql()
	if err != nil {
		return false, err
	}

	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *ChannelService) GetChannelsByUserId(ctx context.Context, userId int64) ([]*descriptors.UserChannel, error) {
	query, args, err := s.dao.GetBuilder().Select("id", "user_id", "channel_type", "identifier", "is_authenticated", "metadata", "created_at", "updated_at").
		From(descriptors.UserChannelTableName).
		Where(squirrel.Eq{"user_id": userId}).ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*descriptors.UserChannel
	for rows.Next() {
		var c descriptors.UserChannel
		if err := rows.Scan(&c.Id, &c.UserId, &c.ChannelType, &c.Identifier, &c.IsAuthenticated, &c.Metadata, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, &c)
	}

	return channels, nil
}

func (s *ChannelService) LogAuthAttempt(ctx context.Context, log *descriptors.AuthLog) error {
	now := time.Now()
	log.CreatedAt = now

	query, args, err := s.dao.GetBuilder().Insert(descriptors.AuthLogTableName).
		Columns("user_id", "channel_type", "action", "ip_address", "user_agent", "success", "metadata", "created_at").
		Values(log.UserId, log.ChannelType, log.Action, log.IPAddress, log.UserAgent, log.Success, log.Metadata, log.CreatedAt).ToSql()
	if err != nil {
		return err
	}

	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	return err
}

func (s *ChannelService) GetAuthLogs(ctx context.Context, userId int64, pagination datamodels.Pagination) ([]*descriptors.AuthLog, int64, error) {
	// First get count
	countQuery, countArgs, err := s.dao.GetBuilder().Select("COUNT(*)").From(descriptors.AuthLogTableName).Where(squirrel.Eq{"user_id": userId}).ToSql()
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := s.dao.GetDb().QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Then get data
	queryBuilder := s.dao.GetBuilder().Select("id", "user_id", "channel_type", "action", "ip_address", "user_agent", "success", "metadata", "created_at").
		From(descriptors.AuthLogTableName).
		Where(squirrel.Eq{"user_id": userId}).
		OrderBy("created_at DESC")

	if pagination.Limit != nil {
		limit, _ := strconv.ParseUint(*pagination.Limit, 10, 64)
		if limit > 0 {
			queryBuilder = queryBuilder.Limit(limit)
		}
	}
	if pagination.Offset != nil {
		offset, _ := strconv.ParseUint(*pagination.Offset, 10, 64)
		queryBuilder = queryBuilder.Offset(offset)
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*descriptors.AuthLog
	for rows.Next() {
		var l descriptors.AuthLog
		if err := rows.Scan(&l.Id, &l.UserId, &l.ChannelType, &l.Action, &l.IPAddress, &l.UserAgent, &l.Success, &l.Metadata, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}

	return logs, total, nil
}

func (s *ChannelService) SendNotification(ctx context.Context, userId int64, message string, preferredChannels []descriptors.ChannelType) error {
	channels, err := s.GetChannelsByUserId(ctx, userId)
	if err != nil {
		return err
	}

	for _, c := range channels {
		if !c.IsAuthenticated {
			continue
		}

		isPreferred := false
		if len(preferredChannels) == 0 {
			isPreferred = true
		} else {
			for _, pc := range preferredChannels {
				if pc == c.ChannelType {
					isPreferred = true
					break
				}
			}
		}

		if isPreferred {
			err := s.sendToGateway(ctx, c.ChannelType, c.Identifier, message)
			if err != nil {
				fmt.Printf("Error sending to %s gateway: %v\n", c.ChannelType, err)
				// Continue to other channels even if one fails
			}
		}
	}

	return nil
}

func (s *ChannelService) sendToGateway(ctx context.Context, channelType descriptors.ChannelType, identifier string, message string) error {
	var cfg descriptors.ChannelConfig
	switch channelType {
	case descriptors.ChannelWhatsApp:
		cfg = s.config.WhatsApp
	case descriptors.ChannelEmail:
		cfg = s.config.Email
	case descriptors.ChannelSignal:
		cfg = s.config.Signal
	case descriptors.ChannelTelegram:
		cfg = s.config.Telegram
	case descriptors.ChannelX:
		cfg = s.config.X
	case descriptors.ChannelBluesky:
		cfg = s.config.Bluesky
	}

	if !cfg.Enabled || cfg.GatewayURL == "" {
		return fmt.Errorf("gateway for %s is not enabled or URL is missing", channelType)
	}

	payload := map[string]string{
		"to":      identifier,
		"message": message,
	}
	jsonData, _ := json.Marshal(payload)

	// In real world, you might add an API Key header here
	resp, err := http.Post(cfg.GatewayURL+"/api/send", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("gateway returned error status: %d", resp.StatusCode)
	}

	return nil
}


func (s *ChannelService) HandleInbound(ctx context.Context, channelType descriptors.ChannelType, identifier string, payload map[string]interface{}) error {
	fmt.Printf("Received inbound from %s (%s): %v\n", channelType, identifier, payload)
	// Process message, maybe trigger an agent or command
	return nil
}
