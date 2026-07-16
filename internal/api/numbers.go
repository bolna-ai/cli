package api

// PhoneNumber is one row of GET /phone-numbers/all.
type PhoneNumber struct {
	ID                string `json:"id"`
	PhoneNumber       string `json:"phone_number"`
	AgentID           string `json:"agent_id"`
	TelephonyProvider string `json:"telephony_provider"`
	Rented            bool   `json:"rented"`
	Price             string `json:"price"`
	CreatedAt         string `json:"created_at"`
	RenewalAt         string `json:"renewal_at"`
}

func (c *Client) ListPhoneNumbers() ([]PhoneNumber, error) {
	var numbers []PhoneNumber
	err := c.do("/phone-numbers/all", requestOptions{}, &numbers)
	return numbers, err
}
