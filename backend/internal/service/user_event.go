package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type UserEventType string

const (
	EventProfileUpdated  UserEventType = "profile.updated"
	EventEmailVerified   UserEventType = "email.verified"
	EventStatusChanged   UserEventType = "status.changed"
	EventPasswordChanged UserEventType = "password.changed"
)

type UserEventData struct {
	Nickname        string     `json:"nickname,omitempty"`
	Bio             string     `json:"bio,omitempty"`
	Avatar          string     `json:"avatar,omitempty"`
	Email           string     `json:"email,omitempty"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	Status          string     `json:"status,omitempty"`
}

type UserEvent struct {
	EventID   string        `json:"event_id"`
	EventType UserEventType `json:"event_type"`
	UserID    uint          `json:"user_id"`
	Timestamp time.Time     `json:"timestamp"`
	Data      UserEventData `json:"data"`
}

type UserEventPublisher struct {
	redis *redis.Client
}

func NewUserEventPublisher(rdb *redis.Client) *UserEventPublisher {
	return &UserEventPublisher{redis: rdb}
}

func (p *UserEventPublisher) Publish(ctx context.Context, eventType UserEventType, userID uint, data UserEventData) {
	if p.redis == nil {
		return
	}
	event := UserEvent{
		EventID:   uuid.New().String(),
		EventType: eventType,
		UserID:    userID,
		Timestamp: time.Now(),
		Data:      data,
	}
	payload, _ := json.Marshal(event)
	if err := p.redis.Publish(ctx, "sso:user:updated", payload).Err(); err != nil {
		zap.L().Warn("failed to publish user event",
			zap.String("event_type", string(eventType)),
			zap.Uint("user_id", userID),
			zap.Error(err))
	}
}
