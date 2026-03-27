package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
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
		Id:              now.UnixNano(), // Temporary ID since we don't have auto-increment in IPrimaryDao
		UserId:          userId,
		ChannelType:     channelType,
		Identifier:      identifier,
		IsAuthenticated: false,
		Metadata:        string(metadataJson),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	key := fmt.Sprintf("%d:%s", userId, identifier)
	rec := datamodels.RecJSON{
		Namespace: descriptors.UserChannelTableName,
		Key:       key,
		Rec:       userChannel,
		Tmstamp:   now,
	}

	err := s.dao.Save(ctx, rec)
	if err != nil {
		return nil, err
	}

	return userChannel, nil
}

func (s *ChannelService) VerifyChannel(ctx context.Context, userId int64, channelType descriptors.ChannelType, token string) (bool, error) {
	// Fetch channels for this user and type
	filters := []datamodels.Filter{
		{
			FieldName: "userId",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{userId}},
			},
		},
		{
			FieldName: "channelType",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{channelType}},
			},
		},
	}

	recs, _, err := s.dao.List(ctx, descriptors.UserChannelTableName, filters, datamodels.Pagination{}, nil)
	if err != nil {
		return false, err
	}

	for _, r := range recs {
		// Update each matching channel (usually only one)
		var c descriptors.UserChannel
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &c)
		
		c.IsAuthenticated = true
		c.UpdatedAt = time.Now()
		
		r.Rec = c
		r.Tmstamp = c.UpdatedAt
		if err := s.dao.Save(ctx, r); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (s *ChannelService) GetChannelsByUserId(ctx context.Context, userId int64) ([]*descriptors.UserChannel, error) {
	filters := []datamodels.Filter{
		{
			FieldName: "userId",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{userId}},
			},
		},
	}

	recs, _, err := s.dao.List(ctx, descriptors.UserChannelTableName, filters, datamodels.Pagination{}, nil)
	if err != nil {
		return nil, err
	}

	var channels []*descriptors.UserChannel
	for _, r := range recs {
		var c descriptors.UserChannel
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &c)
		channels = append(channels, &c)
	}

	return channels, nil
}

func (s *ChannelService) LogAuthAttempt(ctx context.Context, log *descriptors.AuthLog) error {
	now := time.Now()
	log.CreatedAt = now
	log.Id = now.UnixNano()

	key := fmt.Sprintf("%d", log.Id)
	if log.UserId != nil {
		key = fmt.Sprintf("%d:%d", *log.UserId, log.Id)
	}

	rec := datamodels.RecJSON{
		Namespace: descriptors.AuthLogTableName,
		Key:       key,
		Rec:       log,
		Tmstamp:   now,
	}

	return s.dao.Save(ctx, rec)
}

func (s *ChannelService) GetAuthLogs(ctx context.Context, userId int64, pagination datamodels.Pagination) ([]*descriptors.AuthLog, int64, error) {
	filters := []datamodels.Filter{
		{
			FieldName: "userId",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{userId}},
			},
		},
	}

	sorts := []datamodels.Sort{
		{Field: "createdAt", Order: datamodels.SortOrderDesc},
	}

	recs, total, err := s.dao.List(ctx, descriptors.AuthLogTableName, filters, pagination, sorts)
	if err != nil {
		return nil, 0, err
	}

	var logs []*descriptors.AuthLog
	for _, r := range recs {
		var l descriptors.AuthLog
		data, _ := json.Marshal(r.Rec)
		json.Unmarshal(data, &l)
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
