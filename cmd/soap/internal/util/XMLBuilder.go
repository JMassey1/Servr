package util

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

type EnvelopeBuilder struct {
	xmlns  string
	header interface{}
	body   interface{}
	indent string
}

func NewEnvelopeBuilder() *EnvelopeBuilder {
	return &EnvelopeBuilder{}
}

func (builder *EnvelopeBuilder) WithDefaultNamespace() *EnvelopeBuilder {
	builder.xmlns = "http://schemas.xmlsoap.org/soap/envelope/"
	return builder
}

func (builder *EnvelopeBuilder) WithNamespace(namespace string) *EnvelopeBuilder {
	builder.xmlns = namespace
	return builder
}

func (builder *EnvelopeBuilder) WithHeader(header interface{}) *EnvelopeBuilder {
	builder.header = header
	return builder
}

func (builder *EnvelopeBuilder) WithBody(body interface{}) *EnvelopeBuilder {
	builder.body = body
	return builder
}

func (builder *EnvelopeBuilder) WithIndent(prefix, indent string) *EnvelopeBuilder {
	builder.indent = prefix + "\n" + indent
	return builder
}

func (builder *EnvelopeBuilder) Build() ([]byte, error) {
	wrapper := struct {
		XMLName xml.Name    `xml:"soap:Envelope"`
		Xmlns   string      `xml:"xmlns:soap,attr"`
		Header  interface{} `xml:"soap:Header,omitempty"`
		Body    interface{} `xml:"soap:Body"`
	}{
		Xmlns:  builder.xmlns,
		Header: builder.header,
		Body:   builder.body,
	}

	var (
		output []byte
		err    error
	)
	if builder.indent == "" {
		output, err = xml.Marshal(wrapper)
	} else {
		buffer := &bytes.Buffer{}
		buffer.WriteString(xml.Header)

		encoder := xml.NewEncoder(buffer)
		encoder.Indent("", "  ")
		if err = encoder.Encode(wrapper); err != nil {
			return nil, err
		}
		return buffer.Bytes(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("soap build: %w", err)
	}
	return append([]byte(xml.Header), output...), nil
}
