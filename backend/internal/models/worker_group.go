package models

import (
	"time"

	"github.com/google/uuid"
)

type WorkerGroup struct {
	GroupID   uuid.UUID `db:"group_id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}
