package audit

import "context"

// ActionType описывает типы аудита, отправляемые через провайдера.
type ActionType string

const (
	// AuditActionTypeShorten используется при событии создания сокращения.
	AuditActionTypeShorten ActionType = "shorten"
	// AuditActionTypeFollow используется при переходе на ссылку.
	AuditActionTypeFollow ActionType = "follow"
)

// Action представляет событие, записываемое в аудит-лог.
type Action struct {
	// Timestamp это числовое представление даты и времени (Unix timestamp).
	Timestamp int64 `json:"ts"`
	// ActionType это тип произошедшего события.
	ActionType ActionType `json:"action"`
	// UserID это ID пользователя, совершившего событие, если он есть.
	UserID *string `json:"user_id,omitempty"`
	// URL это оригинальный (не сокращённый) URL для события.
	URL string `json:"url"`
}

// Provider описывает интерфейс провайдера аудит-системы.
type Provider interface {
	// Send отправляет новое событие аудита через провайдера.
	Send(ctx context.Context, actionType ActionType, userID *string, url string) error
	// Close закрывает данного провайдера (graceful shutdown).
	Close() error
}
