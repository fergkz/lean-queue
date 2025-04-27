package DomainRepositories

import (
	DomainEntities "lean-queue/src/domain/entities"
	"time"
)

type QueueRepositoryInterface interface {
	Save(message DomainEntities.QueueEntity) error
	GetById(id string) (*DomainEntities.QueueEntity, error)
	GetAndReserveMessages(
		queueName DomainEntities.QueueNameEntity,
		limit int,
		messagesBefore time.Time,
		updateReservedAt time.Time,
		updateReservedBy string,
		updateReservedInfo *string,
		updateReservedExpires *time.Time,
	) ([]DomainEntities.QueueEntity, error)
}
