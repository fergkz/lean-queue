package ApplicationUsecases

import (
	DomainEntities "lean-queue/src/domain/entities"
	DomainRepositories "lean-queue/src/domain/repositories"
	"time"
)

type getAndReserveNextMessagesUsecase struct {
	queueRepository DomainRepositories.QueueRepositoryInterface
}

func NewGetAndReserveNextMessagesUsecase(
	queueRepository DomainRepositories.QueueRepositoryInterface,
) *getAndReserveNextMessagesUsecase {
	return &getAndReserveNextMessagesUsecase{
		queueRepository: queueRepository,
	}
}

func (usecase *getAndReserveNextMessagesUsecase) Handle(queueName string, limit int, reservedBy string, reserveBySeconds int, reservedInfo *string) ([]DomainEntities.QueueEntity, error) {

	if *reservedInfo == "" {
		reservedInfo = nil
	}

	queueNameEntity, err := DomainEntities.NewQueueName(queueName)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(time.Duration(reserveBySeconds) * time.Second)

	messages, err := usecase.queueRepository.GetAndReserveMessages(
		*queueNameEntity,
		limit,
		time.Now(),
		time.Now(),
		reservedBy,
		reservedInfo,
		&expiresAt,
	)

	if err != nil {
		return nil, err
	}

	return messages, nil
}
