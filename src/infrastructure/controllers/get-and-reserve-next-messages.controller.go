package InfrastructureControllers

import (
	"encoding/json"
	ApplicationUsecases "lean-queue/src/application/usecases"
	DomainRepositories "lean-queue/src/domain/repositories"
	"log"
	"net/http"
	"strconv"
)

type getAndReserveNextMessagesController struct {
	queueRepository DomainRepositories.QueueRepositoryInterface
}

func NewGetAndReserveNextMessagesController(
	queueRepository DomainRepositories.QueueRepositoryInterface,
) *getAndReserveNextMessagesController {
	return &getAndReserveNextMessagesController{
		queueRepository: queueRepository,
	}
}

func (controller *getAndReserveNextMessagesController) Handle(w http.ResponseWriter, r *http.Request) {
	usecase := ApplicationUsecases.NewGetAndReserveNextMessagesUsecase(
		controller.queueRepository,
	)

	var queueName string = r.URL.Query().Get("queue_name")
	var limitStr string = r.URL.Query().Get("limit")
	limitInt, _ := strconv.Atoi(limitStr)
	var reservedBy string = r.URL.Query().Get("reserved_by")
	var reservedInfo string = r.URL.Query().Get("reserved_info")
	var reserveBySeconds int = 60
	if r.URL.Query().Get("reserve_by_seconds") != "" {
		reserveBySeconds, _ = strconv.Atoi(r.URL.Query().Get("reserve_by_seconds"))
	}

	if queueName == "" {
		http.Error(w, "Missing queue_name parameter", http.StatusBadRequest)
		return
	}

	if limitInt == 0 {
		limitStr = "1"
		limitInt = 1
	}

	if reservedBy == "" {
		http.Error(w, "Missing reserved_by parameter", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		http.Error(w, "Invalid limit parameter: "+err.Error(), http.StatusBadRequest)
		return
	}

	messages, err := usecase.Handle(queueName, limit, reservedBy, reserveBySeconds, &reservedInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Reserved %+v", messages)

	outputObject := make([]map[string]interface{}, len(messages))
	for i, message := range messages {
		outputObject[i] = map[string]interface{}{
			"id":              message.GetId(),
			"queue_name":      message.GetName().GetValue(),
			"message":         message.GetMessage().GetValue(),
			"published_at":    message.GetPublishedAt().UTC().Format("2006-01-02 15:04:05.999999"),
			"reserved_at":     message.GetReservedAt().UTC().Format("2006-01-02 15:04:05.999999"),
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
