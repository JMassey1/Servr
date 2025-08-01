package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	SOAP "mock-server/cmd/soap/internal/models"
	builder "mock-server/cmd/soap/internal/util"
	"mock-server/cmd/soap/internal/wsdl"
	GlobalModels "mock-server/internal/common/models"
	"mock-server/internal/consts"

	CharmLog "github.com/charmbracelet/log"
)

var logger = CharmLog.NewWithOptions(os.Stderr, CharmLog.Options{
	ReportTimestamp: true,
	TimeFormat:      time.Kitchen,
	Prefix:          "SOAP Service 🧼",
})

var mockCustomers = []GlobalModels.Customer{
	{ID: 1, Name: "Alice Smith", Cust_Type: "Regular", Email: "alicsmith@example.com"},
	{ID: 2, Name: "Bob Johnson", Cust_Type: "Premium", Email: "bobjohnson22@example.net"},
	{ID: 3, Name: "Charlie Brown", Cust_Type: "Regular", Email: "cbrown_und3r@example.com"},
	{ID: 4, Cust_Type: "Closed"},
}

func setupSOAPServer() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/soap?wsdl", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(wsdl.GetWSDL()))
	})

	mux.HandleFunc("/soap", parseSOAPRequest)
	return mux
}

func parseSOAPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Only POST Allowed", http.StatusMethodNotAllowed)
		return
	}

	// read the raw request body
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		sendSOAPFault(w, "Client", "could not read request body", err)
		return
	}

	// extract everything under <soap:Body>
	var envelope struct {
		Body struct {
			InnerXML []byte `xml:",innerxml"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal(raw, &envelope); err != nil {
		sendSOAPFault(w, "Client", "Malformed SOAP envelope", err)
		return
	}

	// determine payload/sub-request
	switch {
	case bytes.Contains(envelope.Body.InnerXML, []byte("<Echo>")):
		logger.Info("Recieved EchoRequest")
		var request SOAP.EchoRequest
		if err := xml.Unmarshal(envelope.Body.InnerXML, &request); err != nil {
			sendSOAPFault(w, "Client", "Bad EchoRequest", err)
			return
		}

		responsePayload := SOAP.EchoResponse{Message: request.Message}
		out, err := builder.
			NewEnvelopeBuilder().
			WithDefaultNamespace().
			WithBody(responsePayload).
			Build()
		if err != nil {
			sendSOAPFault(w, "Server", "Could not build SOAP response", err)
			return
		}
		logger.With("method", "EchoRquest").Info(request.Message)
		w.Header().Set("Content-Type", "text/xml")
		w.Write(out)

	case bytes.Contains(envelope.Body.InnerXML, []byte("<GetCustomer>")):
		logger.Info("Received GetCustomerRequest")
		var request SOAP.GetCustomerRequest
		if err := xml.Unmarshal(envelope.Body.InnerXML, &request); err != nil {
			sendSOAPFault(w, "Client", "Bad GetCustomerRequest", err)
			return
		}

		customer, err := func(id int) (GlobalModels.Customer, error) {
			for _, c := range mockCustomers {
				if c.ID == id {
					return c, nil
				}
			}
			return GlobalModels.Customer{}, fmt.Errorf("customer with id %d not found", id)
		}(request.CustomerID)
		if err != nil {
			sendSOAPFault(w, "Server", "Customer not found", err)
			return
		}

		responsePayload := SOAP.GetCustomerResponse{Customer: customer}
		out, err := builder.
			NewEnvelopeBuilder().
			WithDefaultNamespace().
			WithBody(responsePayload).
			Build()
		if err != nil {
			sendSOAPFault(w, "Server", "Could not build SOAP response", err)
			return
		}
		logger.Info("Responding to request", "customerId", customer.ID, "customerName", customer.Name)
		w.Header().Set("Content-Type", "text/xml")
		w.Write(out)
	default:
		sendSOAPFault(w, "Client", "Unkown request operation", nil)
	}
}

func sendSOAPFault(w http.ResponseWriter, code, message string, err error) {
	fault := builder.MakeFaultMessage(code, message)
	w.Header().Set("Content-Type", "text/xml")

	if code == "Client" {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

	logger.Error("Error parsing SOAP request", "message", message, "error", err)
	w.Write([]byte(fault))
}

func main() {
	handler := setupSOAPServer()

	logger.Info(fmt.Sprintf("Listening on port %d", consts.SOAP_PORT))
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", consts.SOAP_PORT), handler))
}
