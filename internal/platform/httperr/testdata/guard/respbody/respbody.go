// Package respbody reads resp.Body in a file that does not import problem; the
// guard must pass it.
package respbody

import (
	"io"
	"net/http"
)

func Fetch(c *http.Client, req *http.Request) ([]byte, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
