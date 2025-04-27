package InfrastructureControllers

import (
	"encoding/json"
	ApplicationUsecases "lean-queue/src/application/usecases"
	DomainRepositories "lean-queue/src/domain/repositories"
	"net/http"
)

type removeMessageController struct {
	queueRepository DomainRepositories.QueueRepositoryInterface
}

func NewRemoveMessageController(
	queueRepository DomainRepositories.QueueRepositoryInterface,
) *removeMessageController {
	return &removeMessageController{
		queueRepository: queueRepository,
	}
}

func (controller *removeMessageController) Handle(w http.ResponseWriter, r *http.Request) {
	usecase := ApplicationUsecases.NewRemoveMessageUsecase(
		controller.queueRepository,
	)

	type requestBody struct {
		MessageId string `json:"message_id"`
	}

	var body requestBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Erro ao ler o corpo da requisição: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	err = usecase.Handle(body.MessageId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message removed successfully"))
}
