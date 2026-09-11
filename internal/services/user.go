package services

import "context"

type (
	userIdKey string
)

const (
	userKey userIdKey = "user_id"
)

func (s *Service) SetUserIdToCtx(ctx context.Context, userId string) context.Context {
	return context.WithValue(ctx, userKey, userId)
}

func (s *Service) GetUserIdFromCtx(ctx context.Context) string {
	userId, ok := ctx.Value(userKey).(string)

	if !ok {
		return ""
	}

	return userId
}
