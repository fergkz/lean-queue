package ApplicationUsecases

import (
	DomainEntities "lean-queue/src/domain/entities"
	DomainRepositories "lean-queue/src/domain/repositories"
)

type getMessagesOnQueueUsecase struct {
	queueRepository DomainRepositories.QueueRepositoryInterface
}

func NewGetMessagesOnQueueUsecase(
	queueRepository DomainRepositories.QueueRepositoryInterface,
) *getMessagesOnQueueUsecase {
	return &getMessagesOnQueueUsecase{
		queueRepository: queueRepository,
	}
}

func (usecase *getMessagesOnQueueUsecase) Handle(queueName string, limit int) ([]DomainEntities.QueueEntity, error) {

	queueNameEntity, err := DomainEntities.NewQueueName(queueName)
	if err != nil {
		return nil, err
	}

	messages, err := usecase.queueRepository.GetMessages(
		*queueNameEntity,
		limit,
	)

	if err != nil {
		return nil, err
	}

	return messages, nil
}
