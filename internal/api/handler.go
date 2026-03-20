package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type TaskServer struct {
	api ApiInterface
}

func NewTaskServer(api ApiInterface) *TaskServer {
	return &TaskServer{api: api}
}
func (s *TaskServer) CreateUsersPost(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Server is running"))
}

func SetupRouter(api ApiInterface, logger *zap.Logger) *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/create_user", api.CreateUsersPost).Methods("POST")
	router.HandleFunc("/api/weather", api.GetWeather).Methods(http.MethodGet)

	return router

}
