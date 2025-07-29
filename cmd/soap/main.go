package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	M "mock-server/internal/common/models"
	"mock-server/internal/consts"

	CharmLog "github.com/charmbracelet/log"
)

var logger = CharmLog.NewWithOptions(os.Stderr, CharmLog.Options{
	ReportTimestamp: true,
	TimeFormat:      time.Kitchen,
	Prefix:          "REST Service📡",
})

var mockCustomers = []M.Customer{
	{ID: 1, Name: "Alice Smith", Cust_Type: "Regular", Email: "alicsmith@example.com"},
	{ID: 2, Name: "Bob Johnson", Cust_Type: "Premium", Email: "bobjohnson22@example.net"},
	{ID: 3, Name: "Charlie Brown", Cust_Type: "Regular", Email: "cbrown_und3r@example.com"},
	{ID: 4, Cust_Type: "Closed"},
}

type SOAPEnvelope struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	Xmlns   string   `xml:"xmlns:soap,attr"`
	Body    SOAPBody `xml:"soap:Body"`
}

type SOAPBody struct {
	Content any `xml:",any"`
}

type SOAPFault struct {
	XMLName xml.Name `xml:"soap:Fault"`
	Code    string   `xml:"faultcode"`
	String  string   `xml:"faultstring"`
}

type SOAPEchoRequest struct {
	XMLName xml.Name `xml:"Echo"`
	Message string   `xml:"message"`
}
type SOAPEchoResponse struct {
	XMLName xml.Name `xml:"EchoResponse"`
	Message string   `xml:"message"`
}

type GetCustomerRequest struct {
	XMLName    xml.Name `xml:"GetCustomer"`
	CustomerID int      `xml:"customerId"`
}
type GetCustomerResponse struct {
	XMLName  xml.Name   `xml:"GetCustomerResponse"`
	Customer M.Customer `xml:"customer"`
}

func setupSOAPServer() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/soap?wsdl", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(getWSDL()))
	})

	mux.HandleFunc("/soap", handleSOAPRequest)
	return mux
}

func handleSOAPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendSOAPFault(w, "Client", "Could not read request body")
		return
	}

	output, err := xml.MarshalIndent(string(body), "", "  ")
	if err != nil {
		sendSOAPFault(w, "Server", "Failed to marshal request body")
		return
	}
	logger.Debug("SOAP Request: %s", output)

	var envelope SOAPEnvelope
	if err := xml.Unmarshal(body, &envelope); err != nil {
		sendSOAPFault(w, "Client", "Malformed SOAP envelope")
		return
	}

	if strings.Contains(string(body), "<Echo>") {
		var echoReq SOAPEchoRequest
		if err := xml.Unmarshal([]byte(envelope.Body.Content.(string)), &echoReq); err != nil {
			sendSOAPFault(w, "Client", "Malformed Echo request")
			return
		}

		logger.Info("SOAP -- Echo: '%s'", echoReq.Message)

		resp := SOAPEchoResponse{Message: echoReq.Message}
		respEnvelope := SOAPEnvelope{
			Xmlns: "http://schemas.xmlsoap.org/soap/envelope/",
			Body:  SOAPBody{Content: resp},
		}

		output, err := xml.MarshalIndent(respEnvelope, "", "  ")
		if err != nil {
			sendSOAPFault(w, "Server", "Failed to marshal response")
			return
		}

		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xml.Header + string(output)))
		return
	} else if strings.Contains(string(body), "<GetCustomer>") {
		var getCustReq GetCustomerRequest
		if err := xml.Unmarshal([]byte(envelope.Body.Content.(string)), &getCustReq); err != nil {
			sendSOAPFault(w, "Client", "Malformed GetCustomer request")
			return
		}

		logger.Info("SOAP -- GetCustomer: ID %d", getCustReq.CustomerID)

		customer, err := func(id int) (M.Customer, error) {
			for _, c := range mockCustomers {
				if c.ID == id {
					return c, nil
				}
			}
			return M.Customer{}, fmt.Errorf("customer with id %d not found", id)
		}(getCustReq.CustomerID)
		if err != nil {
			sendSOAPFault(w, "Server", err.Error())
			return
		}

		resp := GetCustomerResponse{Customer: customer}
		respEnvelope := SOAPEnvelope{
			Xmlns: "http://schemas.xmlsoap.org/soap/envelope/",
			Body: SOAPBody{
				Content: resp,
			},
		}

		output, err := xml.MarshalIndent(respEnvelope, "", "  ")
		if err != nil {
			sendSOAPFault(w, "Server", "Failed to marshal response")
			return
		}

		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xml.Header + string(output)))
		return
	} else {
		sendSOAPFault(w, "Client", "Unknown operation")
	}
}

func sendSOAPFault(w http.ResponseWriter, code, message string) {
	fault := `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <soap:Fault>
      <faultcode>` + code + `</faultcode>
      <faultstring>` + message + `</faultstring>
    </soap:Fault>
  </soap:Body>
</soap:Envelope>`

	w.Header().Set("Content-Type", "text/xml")

	if code == "Client" {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

	logger.Error("SOAP Fault: %s", fault)
	w.Write([]byte(fault))
}

func getWSDL() string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<definitions name="MockSOAPService"
    targetNamespace="http://example.com/soap"
    xmlns:tns="http://example.com/soap"
    xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema"
    xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"
    xmlns="http://schemas.xmlsoap.org/wsdl/">

    <types>
        <xsd:schema targetNamespace="http://example.com/soap">
            <xsd:element name="Echo">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="message" type="xsd:string" />
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>

            <xsd:element name="GetCustomer">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="customerId" type="xsd:int" />
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>

            <!-- Optional: Responses -->
            <xsd:element name="EchoResponse" type="xsd:string" />
            <xsd:element name="GetCustomerResponse" type="xsd:string" />
        </xsd:schema>
    </types>

    <message name="EchoRequest">
        <part name="parameters" element="tns:Echo" />
    </message>
    <message name="EchoResponse">
        <part name="parameters" element="tns:EchoResponse" />
    </message>

    <message name="GetCustomerRequest">
        <part name="parameters" element="tns:GetCustomer" />
    </message>
    <message name="GetCustomerResponse">
        <part name="parameters" element="tns:GetCustomerResponse" />
    </message>

    <portType name="MockSOAPPortType">
        <operation name="Echo">
            <input message="tns:EchoRequest" />
            <output message="tns:EchoResponse" />
        </operation>
        <operation name="GetCustomer">
            <input message="tns:GetCustomerRequest" />
            <output message="tns:GetCustomerResponse" />
        </operation>
    </portType>

    <binding name="MockSOAPBinding" type="tns:MockSOAPPortType">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http" />
        <operation name="Echo">
            <soap:operation soapAction="http://example.com/soap/Echo" />
            <input>
                <soap:body use="literal" />
            </input>
            <output>
                <soap:body use="literal" />
            </output>
        </operation>
        <operation name="GetCustomer">
            <soap:operation soapAction="http://example.com/soap/GetCustomer" />
            <input>
                <soap:body use="literal" />
            </input>
            <output>
                <soap:body use="literal" />
            </output>
        </operation>
    </binding>

    <service name="MockSOAPService">
        <port name="MockSOAPPort" binding="tns:MockSOAPBinding">
            <soap:address location="http://localhost:%d/soap" />
        </port>
    </service>
</definitions>`, consts.SOAP_PORT)
}

func main() {
	handler := setupSOAPServer()

	logger.Info("Listening on :%d", consts.SOAP_PORT)
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", consts.SOAP_PORT), handler))
}
