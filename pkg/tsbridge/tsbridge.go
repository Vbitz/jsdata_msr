package tsbridge

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	defaultAddress = flag.String("addr", "localhost:5123", "The address of the backend TypeScript server.")
)

type Request struct {
	Filename     string `json:"filename"`
	FileContents string `json:"fileContents"`
}

type Response struct {
	Version     int             `json:"version"`
	ProcessTime uint64          `json:"processTime"`
	Features    map[string]bool `json:"features"`
}

type Bridge struct {
	addr string
}

func (b *Bridge) Call(req Request) (Response, error) {
	var err error

	left := 10

	for {
		left -= 1
		if left == 0 {
			return Response{}, fmt.Errorf("Call failed: %v", err)
		}

		body := new(bytes.Buffer)

		enc := json.NewEncoder(body)

		err = enc.Encode(&req)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		res, err := http.Post(fmt.Sprintf("http://%s/process", b.addr), "application/json", body)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		var resp Response

		respBody, err := io.ReadAll(res.Body)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		err = json.Unmarshal(respBody, &resp)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		return resp, nil
	}
}

func NewBridge(addr string) *Bridge {
	if addr == "" {
		addr = *defaultAddress
	}

	return &Bridge{addr: addr}
}
