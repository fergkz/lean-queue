package DomainEntities

import (
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
	id            string
	name          QueueNameEntity
	message       QueueMessageEntity
	publishedAt   time.Time
	reservedAt    *time.Time
	reservedBy    *string
	reservedCount *int
	reservedInfo  *string
}

func NewQueue(id *string, name QueueNameEntity, message QueueMessageEntity, publishedAt time.Time, reservedAt *time.Time, reservedBy *string, reservedCount *int, reservedInfo *string) *QueueEntity {

	if id == nil {
		newUuid := uuid.New().String()
		id = &newUuid
	}

	return &QueueEntity{
		id:            *id,
		name:          name,
		message:       message,
		publishedAt:   publishedAt,
		reservedAt:    reservedAt,
		reservedBy:    reservedBy,
		reservedCount: reservedCount,
		reservedInfo:  reservedInfo,
	}
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
