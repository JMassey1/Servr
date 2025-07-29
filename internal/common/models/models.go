package models

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
