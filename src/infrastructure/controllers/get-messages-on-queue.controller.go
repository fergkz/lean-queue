package InfrastructureControllers

import (
	"encoding/json"
	ApplicationUsecases "lean-queue/src/application/usecases"
	DomainRepositories "lean-queue/src/domain/repositories"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type getMessagesOnQueueController struct {
	queueRepository DomainRepositories.QueueRepositoryInterface
}

func NewGetMessagesOnQueueController(
	queueRepository DomainRepositories.QueueRepositoryInterface,
) *getMessagesOnQueueController {
	return &getMessagesOnQueueController{
		queueRepository: queueRepository,
	}
}

func (controller *getMessagesOnQueueController) Handle(w http.ResponseWriter, r *http.Request) {
	usecase := ApplicationUsecases.NewGetMessagesOnQueueUsecase(
		controller.queueRepository,
	)

	vars := mux.Vars(r)
	queueName := vars["queue_name"]

	var limitStr string = r.URL.Query().Get("limit")
	limitInt, _ := strconv.Atoi(limitStr)

	if queueName == "" {
		http.Error(w, "Missing queue_name parameter", http.StatusBadRequest)
		return
	}

	if limitInt == 0 {
		limitStr = "1"
		limitInt = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		http.Error(w, "Invalid limit parameter: "+err.Error(), http.StatusBadRequest)
		return
	}

	messages, err := usecase.Handle(queueName, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	outputObject := make([]map[string]interface{}, len(messages))
	for i, message := range messages {
		var reservedAtStr *string
		if message.GetReservedAt() != nil {
			reservedAtStrC := message.GetReservedAt().UTC().Format("2006-01-02 15:04:05.999999")
			reservedAtStr = &reservedAtStrC
		}

		outputObject[i] = map[string]interface{}{
			"id":              message.GetId(),
			"queue_name":      message.GetName().GetValue(),
			"message":         message.GetMessage().GetValue(),
			"published_at":    message.GetPublishedAt().UTC().Format("2006-01-02 15:04:05.999999"),
			"reserved_at":     reservedAtStr,
			"reserved_by":     message.GetReservedBy(),
			"reserved_count":  message.GetReservedCount(),
			"reserved_info":   message.GetReservedInfo(),
			"reserve_expires": message.GetReserveExpires().UTC().Format("2006-01-02 15:04:05.999999"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(outputObject)
}
