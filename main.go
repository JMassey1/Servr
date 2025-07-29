package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/gorilla/mux"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	SFTP_PORT = 2022
	HTTP_PORT = 8080
	SOAP_PORT = 8081
	SFTP_ROOT = "./sftp-root"
)

//MARK: - Auth

type User struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

func validateAuth(token string) (*User, error) {
	log.Info("Validating auth token: %s", token)

	if token == "valid-token" || token == "testpass" {
		return &User{
			Username: "testuser",
			Email:    "test@example.com",
			Roles:    []string{"admin", "user"},
		}, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ==============================================================
// MARK: - REST Server
// ==============================================================
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type RESTEchoRequest struct {
	Message string `json:"message"`
}

type Customer struct {
	ID        int    `json:"id"`
	Name      string `json:"name,omitempty"`
	Cust_Type string `json:"type"`
	Email     string `json:"email,omitempty"`
}

var mockCustomers = []Customer{
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
		if _, err := validateAuth(token); err != nil {
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
	log.Info("REST -- (POST): Echo request received")

	var response RESTEchoRequest
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "Invalid JSON"})
		return
	}
	log.Info("REST -- Echo: '%s'", response.Message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: response})
}

func getCustomer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	log.Info("REST -- GET (/customer/%d): Fetching customer with ID %d", id, id)

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

// ==============================================================
// MARK: - SOAP Server
// ==============================================================
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
	XMLName  xml.Name `xml:"GetCustomerResponse"`
	Customer Customer `xml:"customer"`
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
	log.Debug("SOAP Request: %s", output)

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

		log.Info("SOAP -- Echo: '%s'", echoReq.Message)

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

		log.Info("SOAP -- GetCustomer: ID %d", getCustReq.CustomerID)

		customer, err := func(id int) (Customer, error) {
			for _, c := range mockCustomers {
				if c.ID == id {
					return c, nil
				}
			}
			return Customer{}, fmt.Errorf("Customer with id %d not found", id)
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

	log.Error("SOAP Fault: %s", fault)
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
</definitions>`, SOAP_PORT)
}

// ==============================================================
// MARK: - SFTP Server
// ==============================================================
type realFS struct{}
type lister []os.FileInfo

func sftpAuthHandler(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	user, err := validateAuth(string(password))
	if err != nil {
		return nil, err
	}

	return &ssh.Permissions{
		Extensions: map[string]string{
			"user":  user.Username,
			"email": user.Email,
		},
	}, nil
}

// Mock File System Implementation
func (fs *realFS) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	fullPath := filepath.Join(SFTP_ROOT, r.Filepath)
	log.Info("SFTP Read: %s", fullPath)
	return os.Open(fullPath)
}

func (fs *realFS) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	fullPath := filepath.Join(SFTP_ROOT, r.Filepath)
	log.Info("SFTP Write: %s", fullPath)
	return os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func (fs *realFS) Filecmd(r *sftp.Request) error {
	fullPath := filepath.Join(SFTP_ROOT, r.Filepath)
	log.Info("SFTP Command: %s %s", r.Method, fullPath)

	switch r.Method {
	case "Setstat", "Rename":
		return nil //no-op
	case "Remove":
		return os.Remove(fullPath)
	case "Mkdir":
		return os.Mkdir(fullPath, 0755)
	case "Rmdir":
		return os.RemoveAll(fullPath)
	default:
		return fmt.Errorf("unsupported method: %s", r.Method)
	}
}

func (fs *realFS) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	fullPath := filepath.Join(SFTP_ROOT, r.Filepath)
	log.Info("SFTP List: %s", fullPath)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var fileInfos []os.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Printf("SFTP List: Error reading entry %s: %v", entry.Name(), err)
			continue
		}
		fileInfos = append(fileInfos, info)
	}

	return lister(fileInfos), nil
}

// Mock Lister Implementation
func (l lister) ListAt(f []os.FileInfo, off int64) (int, error) {
	if off >= int64(len(l)) {
		return 0, io.EOF
	}

	n := copy(f, l[off:])
	if int(off)+n >= len(l) {
		return n, io.EOF
	}

	return n, nil
}

// MARK: - Main Server
func main() {
	var wg sync.WaitGroup

	log.Printf("Starting mock server...")

	wg.Add(1)
	go func() {
		defer wg.Done()
		startRESTServer()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		startSOAPServer()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		startSFTPServer()
	}()

	wg.Wait()
}

func startRESTServer() {
	router := setupRESTRoutes()

	log.Printf("REST Server listening on :%d", HTTP_PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", HTTP_PORT), router))
}

func startSOAPServer() {
	handler := setupSOAPServer()

	log.Printf("SOAP Server listening on :%d", SOAP_PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", SOAP_PORT), handler))
}

func startSFTPServer() {
	hostKey, err := generateHostKey()
	if err != nil {
		log.Fatal("Failed to generate host key:", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: sftpAuthHandler,
	}
	config.AddHostKey(hostKey)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", SFTP_PORT))
	if err != nil {
		log.Fatal("SFTP: Failed to listen:", err)
	}

	defer listener.Close()
	log.Printf("SFTP Server listening on :%d", SFTP_PORT)
	log.Printf("SFTP Server Details: localhost:%d (user: testuser, pass: testpass)", SFTP_PORT)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("SFTP: Failed to accept connection: %v", err)
			continue
		}

		go handleSFTPConnection(conn, config)
	}
}

func handleSFTPConnection(netConn net.Conn, config *ssh.ServerConfig) {
	defer netConn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, config)
	if err != nil {
		log.Printf("SFTP: SSH handshake failed: %v", err)
		return
	}
	log.Printf("SFTP: New connection from %s", sshConn.RemoteAddr())
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unkown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("SFTP: Could not accept channel: %v", err)
			continue
		}

		go handleSFTPChannel(channel, requests)
	}
}

func handleSFTPChannel(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	for req := range requests {
		if req.Type == "subsystem" && string(req.Payload[4:]) == "sftp" {
			if req.WantReply {
				req.Reply(true, nil)
			}

			handlers := sftp.Handlers{
				FileGet:  &realFS{},
				FilePut:  &realFS{},
				FileCmd:  &realFS{},
				FileList: &realFS{},
			}

			server := sftp.NewRequestServer(channel, handlers)

			log.Printf("SFTP: Session started")
			server.Serve()
			log.Printf("SFTP: Session ended")
			return
		}

		if req.WantReply {
			req.Reply(false, nil)
			log.Printf("SFTP: Unsupported request type: %s", req.Type)
			continue
		}
	}
}

func generateHostKey() (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(filepath.Join(SFTP_ROOT, "ssh", "id_rsa_mockapi"))
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(keyBytes)
}
