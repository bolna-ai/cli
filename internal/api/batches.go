package api

import "net/url"

// Batch is one row of GET /batches/{agent_id}/all.
type Batch struct {
	BatchID     string  `json:"batch_id"`
	Status      string  `json:"status"`
	ScheduledAt *string `json:"scheduled_at"`
	CreatedAt   string  `json:"created_at"`
}

// ListBatches fetches every batch for an agent (bare array, no server-side
// pagination). pageNumber/pageSize > 0 slices client-side for JSON mode;
// pass 0/0 to get everything.
func (c *Client) ListBatches(agentID string, pageNumber, pageSize int) ([]Batch, error) {
	var batches []Batch
	if err := c.do("/batches/"+url.PathEscape(agentID)+"/all", requestOptions{}, &batches); err != nil {
		return nil, err
	}
	if pageNumber > 0 && pageSize > 0 {
		return paginate(batches, pageNumber, pageSize), nil
	}
	return batches, nil
}
