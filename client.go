package easyrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
)

type Client struct {
	// port string
	clt net.Conn
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Connect(port string) error {
	conn, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	c.clt = conn
	return nil
}

func (c *Client) Close() {
	if c.clt != nil {
		c.Close()
	}
}

func (c *Client) Call(server string, fnName string, resp interface{}, params ...interface{}) error {
	ps := make([]interface{}, 0, len(params))
	for _, p := range params {
		ps = append(ps, p)
	}
	b, err := json.Marshal(ps)
	if err != nil {
		return err
	}

	h := &header{
		bodyLength: int64(len(b)),
		server:     server,
		fnName:     fnName,
	}
	_, err = c.clt.Write(c.buildHeader(h))
	if err != nil {
		return err
	}

	_, err = c.clt.Write(b)
	if err != nil {
		return err
	}

	result := make([]byte, headerLength)
	_, err = c.clt.Read(result)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	r, err := c.resolveResult(result)
	if err != nil {
		return err
	}
	b = make([]byte, r.bodyLength)
	_, err = c.clt.Read(b)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	if r.isErr {
		return fmt.Errorf("%s", string(b))
	}
	var v interface{}
	if err = json.Unmarshal(b, &v); err != nil {
		return err
	}
	value, _ := v.([]interface{})
	if len(value) == 0 {
		return nil
	}
	rb, err := json.Marshal(value[0])
	if err != nil {
		return err
	}
	return json.Unmarshal(rb, resp)
}

func (c *Client) buildHeader(h *header) []byte {
	h1 := fmt.Sprintf("%02x", h.bodyLength)
	sfb := []byte(h.server + "." + h.fnName)
	h2 := fmt.Sprintf("%06x", len(sfb))

	hb := make([]byte, 0, headerLength+len(sfb))
	hb = append(hb, []byte(h1)...)
	hb = append(hb, []byte(h2)...)
	hb = append(hb, sfb...)
	return hb
}

type result struct {
	isErr      bool
	bodyLength int64
}

const IsErr = 1

func (c *Client) resolveResult(b []byte) (*result, error) {
	isErr, err := strconv.ParseInt(string(b[0]), 16, 64)
	if err != nil {
		return nil, err
	}
	length, err := strconv.ParseInt(string(b[1:]), 16, 64)
	if err != nil {
		return nil, err
	}

	r := &result{}
	if isErr == IsErr {
		r.isErr = true
	}
	r.bodyLength = length
	return r, nil
}
