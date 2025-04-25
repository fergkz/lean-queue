package InfrastructureControllers

import (
	"encoding/json"
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

	type requestBody struct {
		QueueName string `json:"queue_name"`
		Message   string `json:"message"`
	}

	var body requestBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Erro ao ler o corpo da requisição: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	queueName := body.QueueName
	message := body.Message

	err = usecase.Handle(queueName, message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message published successfully"))
}
