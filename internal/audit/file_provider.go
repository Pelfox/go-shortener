package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	// fileFlags описывает флаги для конечного файла аудита.
	fileFlags = os.O_CREATE | os.O_APPEND | os.O_WRONLY
	// filePermissions описывает права, с которыми будет открыт файл аудита.
	filePermissions = 0o644
)

// generate:reset
// FileProvider реализует провайдера аудита через локальный файл.
type FileProvider struct {
	filePath string
	mutex    sync.Mutex
	entries  []Action
}

// NewFileProvider создаёт и возвращает нового AuditFileProvider.
func NewFileProvider(filePath string) *FileProvider {
	return &FileProvider{
		filePath: filePath,
	}
}

func (p *FileProvider) Send(
	_ context.Context,
	actionType ActionType,
	userID *string,
	url string,
) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.entries = append(p.entries, Action{
		Timestamp:  time.Now().Unix(),
		ActionType: actionType,
		UserID:     userID,
		URL:        url,
	})
	return nil
}

func (p *FileProvider) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.entries) == 0 {
		return nil
	}

	// записываем в режиме JSON lines
	file, err := os.OpenFile(p.filePath, fileFlags, filePermissions)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, entry := range p.entries {
		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("failed to write entry: %w", err)
		}
	}

	p.entries = nil
	return nil
}
