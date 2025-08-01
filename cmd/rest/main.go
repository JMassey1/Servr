package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mock-server/internal/common"
	M "mock-server/internal/common/models"
	"mock-server/internal/consts"

	CharmLog "github.com/charmbracelet/log"
	"github.com/gorilla/mux"
)

var logger = CharmLog.NewWithOptions(os.Stderr, CharmLog.Options{
	ReportTimestamp: true,
	TimeFormat:      time.Kitchen,
	Prefix:          "REST Service📡",
})

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type EchoRequest struct {
	Message string `json:"message"`
}

var mockCustomers = []M.Customer{
	{ID: 1, Name: "Alice Smith", Cust_Type: "Regular", Email: "alicsmith@example.com"},
	{ID: 2, Name: "Bob Johnson", Cust_Type: "Premium", Email: "bobjohnson22@example.net"},
	{ID: 3, Name: "Charlie Brown", Cust_Type: "Regular", Email: "cbrown_und3r@example.com"},
	{ID: 4, Cust_Type: "Closed"},
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		token = strings.TrimPrefix(token, "Bearer ")
		if _, err := common.ValidateAuth(token); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "Authentication Required",
			})
			return
		}

		next(w, r)
	}
}

func setupRESTRoutes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(APIResponse{Success: true, Data: "OK"})
	}).Methods("GET")

	r.HandleFunc("/echo", authMiddleware(echoRequest)).Methods("POST")
	r.HandleFunc("/customer/{id}", authMiddleware(getCustomer)).Methods("GET")

	return r
}

func echoRequest(w http.ResponseWriter, r *http.Request) {
	logger.Info("Echo request received", "method", r.Method, "path", r.URL.Path)

	var response EchoRequest
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		logger.Error("Invalid JSON Receieved", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "Invalid JSON"})
		return
	}
	logger.Info(fmt.Sprintf("Echo: '%s'", response.Message))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: response})
}

func getCustomer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	logger.Info("GetCustomer request received", "customerId", id, "method", r.Method, "path", r.URL.Path)

	for _, customer := range mockCustomers {
		if customer.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(APIResponse{Success: true, Data: customer})
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "Customer not found"})
}

func main() {
	router := setupRESTRoutes()

	logger.Info(fmt.Sprintf("Listening on :%d", consts.HTTP_PORT))
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", consts.HTTP_PORT), router))
}
