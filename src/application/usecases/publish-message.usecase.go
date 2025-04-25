package ApplicationUsecases

import (
	DomainEntities "lean-queue/src/domain/entities"
	DomainRepositories "lean-queue/src/domain/repositories"
	"time"
)

type publishMessageUsecase struct {
	queueRepository DomainRepositories.QueueRepositoryInterface
}

func NewPublishMessageUsecase(
	queueRepository DomainRepositories.QueueRepositoryInterface,
) *publishMessageUsecase {
	return &publishMessageUsecase{
		queueRepository: queueRepository,
	}
}

func (usecase *publishMessageUsecase) Handle(queueName string, message string) error {
	queueNameEntity, err := DomainEntities.NewQueueName(queueName)
	if err != nil {
		return err
	}

	messageEntity, err := DomainEntities.NewQueueMessage(message)
	if err != nil {
		return err
	}

	queueEntity := DomainEntities.NewQueue(nil, *queueNameEntity, *messageEntity, time.Now(), nil, nil, nil, nil)

	err = usecase.queueRepository.Save(*queueEntity)
	if err != nil {
		return err
	}

	return nil
}
