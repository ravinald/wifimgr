package api

import (
	"context"
	"net/http"
)

// fetchAPIData performs a generic GET request to the specified API endpoint
// and returns the raw response data as interface{}
func (c *mistClient) fetchAPIData(ctx context.Context, path string) (interface{}, error) {
	var result interface{}
	err := c.do(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
