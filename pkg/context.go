package pkg

// contextKey - уникальный ключ сервиса для context.Context.
type contextKey string

// ContextUserIDKey это ключ для context.Context, который хранит в себе текущий
// пользовательский ID для запроса.
const ContextUserIDKey contextKey = "userID"
