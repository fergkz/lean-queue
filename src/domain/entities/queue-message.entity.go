package DomainEntities

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type QueueNameEntity struct {
	value string
}

func NewQueueName(value string) (*QueueNameEntity, error) {
	return &QueueNameEntity{value: value}, nil
}

func (qn QueueNameEntity) GetValue() string {
	return qn.value
}

type QueueMessageEntity struct {
	value string
}

func NewQueueMessage(value string) (*QueueMessageEntity, error) {
	return &QueueMessageEntity{value: value}, nil
}

func (qm QueueMessageEntity) GetValue() string {
	return qm.value
}

type QueueEntity struct {
	id             string
	name           QueueNameEntity
	message        QueueMessageEntity
	publishedAt    time.Time
	reservedAt     *time.Time
	reservedBy     *string
	reservedCount  *int
	reservedInfo   *string
	reserveExpires time.Time
}

func NewQueue(
	id *string,
	name QueueNameEntity,
	message QueueMessageEntity,
	publishedAt time.Time,
	reservedAt *time.Time,
	reservedBy *string,
	reservedCount *int,
	reservedInfo *string,
	reserveExpires time.Time,
) (*QueueEntity, error) {

	if id == nil {
		newUuid := uuid.New().String()
		id = &newUuid
	}

	if name.value == "" {
		return nil, errors.New("queue name cannot be empty")
	}

	if message.value == "" {
		return nil, errors.New("queue message cannot be empty")
	}

	if publishedAt.IsZero() {
		return nil, errors.New("publishedAt cannot be zero")
	}

	if reservedAt != nil && reservedAt.IsZero() {
		return nil, errors.New("reservedAt cannot be zero")
	}

	if reservedBy != nil && *reservedBy == "" {
		return nil, errors.New("reservedBy cannot be empty")
	}

	if reservedCount != nil && *reservedCount < 0 {
		return nil, errors.New("reservedCount cannot be negative")
	}

	if reservedInfo != nil && *reservedInfo == "" {
		return nil, errors.New("reservedInfo cannot be empty")
	}

	return &QueueEntity{
		id:             *id,
		name:           name,
		message:        message,
		publishedAt:    publishedAt,
		reservedAt:     reservedAt,
		reservedBy:     reservedBy,
		reservedCount:  reservedCount,
		reservedInfo:   reservedInfo,
		reserveExpires: reserveExpires,
	}, nil
}

func (qm *QueueEntity) GetId() string {
	return qm.id
}

func (qm *QueueEntity) GetName() QueueNameEntity {
	return qm.name
}

func (qm *QueueEntity) GetMessage() QueueMessageEntity {
	return qm.message
}

func (qm *QueueEntity) GetPublishedAt() time.Time {
	return qm.publishedAt
}

func (qm *QueueEntity) GetReservedAt() *time.Time {
	return qm.reservedAt
}

func (qm *QueueEntity) GetReservedBy() *string {
	return qm.reservedBy
}

func (qm *QueueEntity) GetReservedCount() *int {
	return qm.reservedCount
}

func (qm *QueueEntity) GetReservedInfo() *string {
	return qm.reservedInfo
}

func (qm *QueueEntity) GetReserveExpires() time.Time {
	return qm.reserveExpires
}
