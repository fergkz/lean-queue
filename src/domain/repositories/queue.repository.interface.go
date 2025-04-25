package DomainRepositories

import (
	DomainEntities "lean-queue/src/domain/entities"
	"time"
)

type QueueRepositoryInterface interface {
	Save(message DomainEntities.QueueEntity) error
	GetById(id string) (*DomainEntities.QueueEntity, error)
	GetNextByName(name DomainEntities.QueueNameEntity, afterTime time.Time, limit int) ([]DomainEntities.QueueEntity, error)
}
