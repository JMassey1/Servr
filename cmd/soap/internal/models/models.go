package models

import (
	"encoding/xml"

	M "mock-server/internal/common/models"
)

type RequestEnvelope struct {
	XMLName xml.Name    `xml:"Envelope"`
	Xmlns   string      `xml:"soap,attr"`
	Body    RequestBody `xml:"soap:Body"`
}
type RequestBody struct {
	InnerXML []byte `xml:",innerxml"`
}

type ResponseEnvelope struct {
	XMLName xml.Name     `xml:"Envelope"`
	Xmlns   string       `xml:"soap,attr"`
	Body    ResponseBody `xml:"soap:Body"`
}
type ResponseBody struct {
	Content interface{} `xml:",any"`
}

type Fault struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`
	Code    string   `xml:"faultcode"`
	String  string   `xml:"faultstring"`
}

type EchoRequest struct {
	XMLName xml.Name `xml:"Echo"`
	Message string   `xml:"message"`
}
type EchoResponse struct {
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
