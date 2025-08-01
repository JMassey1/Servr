package util

func MakeFaultMessage(code, message string) string {
	return `
	<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	  <soap:Body>
	    <soap:Fault>
	      <faultcode>` + code + `</faultcode>
	      <faultstring>` + message + `</faultstring>
	    </soap:Fault>
	  </soap:Body>
	</soap:Envelope>`
}
