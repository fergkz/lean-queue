package ApplicationUsecases

import (
	DomainRepositories "lean-queue/src/domain/repositories"
)

type removeMessageUsecase struct {
	queueRepository DomainRepositories.QueueRepositoryInterface
}

func NewRemoveMessageUsecase(
	queueRepository DomainRepositories.QueueRepositoryInterface,
) *removeMessageUsecase {
	return &removeMessageUsecase{
		queueRepository: queueRepository,
	}
}

func (usecase *removeMessageUsecase) Handle(messageId string) error {

	err := usecase.queueRepository.RemoveById(messageId)
	if err != nil {
		return err
	}

	return nil
}
