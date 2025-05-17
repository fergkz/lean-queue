package main

import (
	"fmt"
	"io"
	InfrastructureControllers "lean-queue/src/infrastructure/controllers"
	InfrastructureRepositories "lean-queue/src/infrastructure/repositories"
	"log"
	"net/http"
	"net/http/fcgi"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/spf13/viper"
)

func main() {
	currentFormattedTime := time.Now().Format("2006-01")
	logFile, err := os.Create("log-main[" + currentFormattedTime + "].log")
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(multiWriter)
	log.Println("Iniciando...")
	safeGoRoutine(run)
}

func run() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("PANIC/ERROR interno:", err)
			errStr := fmt.Sprintf("%s", err)
			os.WriteFile("debug-LAST-ERROR.txt", []byte(errStr), 0644)
		}
	}()

	runtime.GOMAXPROCS(1)

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file, %s", err)
	}

	config := new(struct {
		MyConfig struct {
			Example string
		}
		Server struct {
			Method  string
			Port    string
			ApiKeys map[string]string
		}
		URL string
	})

	viper.Unmarshal(config)

	router := mux.NewRouter().StrictSlash(true)

	c := cors.New(cors.Options{
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "POST", "PUT", "OPTIONS"},
	})
	handler := c.Handler(router)
	router.StrictSlash(true)

	apiV1Router := router.PathPrefix("/v1").Subrouter()
	apiV1Router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				return
			}

			token := r.Header.Get("ApiAuthorization")

			ok := false
			for _, k := range config.Server.ApiKeys {
				if k == token {
					ok = true
					break
				}
			}

			if ok {
				next.ServeHTTP(w, r)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Token inválido!"))
			}
		})
	})
	apiV1Router.StrictSlash(true)

	repositoryQueue := InfrastructureRepositories.NewQueueRepository(
		viper.GetString("db.host"),
		viper.GetString("db.port"),
		viper.GetString("db.user"),
		viper.GetString("db.password"),
		viper.GetString("db.db_name"),
	)

	controllerPublishMessage := InfrastructureControllers.NewPublishMessageController(repositoryQueue)
	controllerRemoveMessage := InfrastructureControllers.NewRemoveMessageController(repositoryQueue)
	controllerGetAndReserveNextMessages := InfrastructureControllers.NewGetAndReserveNextMessagesController(repositoryQueue)
	controllerGetMessagesOnQueue := InfrastructureControllers.NewGetMessagesOnQueueController(repositoryQueue)

	apiV1Router.HandleFunc("/message", controllerPublishMessage.Handle).Methods("POST")
	apiV1Router.HandleFunc("/message", controllerRemoveMessage.Handle).Methods("DELETE")
	apiV1Router.HandleFunc("/message/next", controllerGetAndReserveNextMessages.Handle).Methods("GET")
	apiV1Router.HandleFunc("/message/queue/{queue_name}", controllerGetMessagesOnQueue.Handle).Methods("GET")

	router.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK - Carregado."))
		},
	).Methods("GET")

	router.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Alive")
		fmt.Fprintf(w, "OK")
	})

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Health")
		fmt.Fprintf(w, "OK")
	})

	if config.Server.Method == "http" {
		log.Printf("Server started at port %s\n", config.Server.Port)
		server := &http.Server{
			Addr:         "0.0.0.0:" + config.Server.Port,
			Handler:      handler,
			ReadTimeout:  120 * time.Second,
			WriteTimeout: 120 * time.Second,
		}
		log.Fatal(server.ListenAndServe())
	} else {
		fcgi.Serve(nil, router)
	}
}

func safeGoRoutine(fn func()) {
	for {
		success := make(chan bool, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					if strErr, ok := r.(string); ok && (strings.Contains(strErr, "pthread_create failed: Resource temporarily unavailable") ||
						strings.Contains(strErr, "unknown pc") ||
						strings.Contains(strErr, "failed to create new OS thread")) {
						log.Println("Erro específico detectado, tentando novamente...")
						success <- false
					} else {
						log.Println("Erro diferente detectado:", r)
						success <- true
					}

					strErr, ok := r.(string)
					if !ok {
						strErr = fmt.Sprintf("%s", r)
						os.WriteFile("debug-LAST-ERROR.txt", []byte(strErr), 0644)
					}
				} else {
					success <- true
				}
			}()
			fn()
		}()

		if <-success {
			break
		}

		time.Sleep(time.Second)
	}
}
