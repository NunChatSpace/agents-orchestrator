package models

import (
	"time"

	"github.com/google/uuid"
)

type MessageRole string

const (
	MessageRoleUser   MessageRole = "user"
	MessageRoleOAgent MessageRole = "oagent"
	MessageRoleWorker MessageRole = "worker"
)

type MessageKind string

const (
	MessageKindInstruction MessageKind = "instruction"
	MessageKindQuestion    MessageKind = "question"
	MessageKindAnswer      MessageKind = "answer"
	MessageKindSummary     MessageKind = "summary"
	MessageKindSystem      MessageKind = "system"
	MessageKindThinking    MessageKind = "thinking"
	MessageKindFileChange  MessageKind = "file_change"
)

type Message struct {
	MessageID uuid.UUID   `db:"message_id"`
	JobID     uuid.UUID   `db:"job_id"`
	Role      MessageRole `db:"role"`
	Kind      MessageKind `db:"kind"`
	Content   string      `db:"content"`
	CreatedAt time.Time   `db:"created_at"`
}
