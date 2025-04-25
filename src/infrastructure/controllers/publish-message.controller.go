package InfrastructureControllers

import (
	ApplicationUsecases "lean-queue/src/application/usecases"
	DomainRepositories "lean-queue/src/domain/repositories"
	"net/http"
)

type publishMessageController struct {
	queueRepository DomainRepositories.QueueRepositoryInterface
}

func NewPublishMessageController(
	queueRepository DomainRepositories.QueueRepositoryInterface,
) *publishMessageController {
	return &publishMessageController{
		queueRepository: queueRepository,
	}
}

func (controller *publishMessageController) Handle(w http.ResponseWriter, r *http.Request) {
	usecase := ApplicationUsecases.NewPublishMessageUsecase(
		controller.queueRepository,
	)

	queueName := r.URL.Query().Get("queue_name")
	message := r.URL.Query().Get("message")

	err := usecase.Handle(queueName, message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message published successfully"))
}
