package wsdl

import (
	"fmt"
	"mock-server/internal/consts"
)

func GetWSDL() string {
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
