package easyrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
)

type Client struct {
	port    string
	clt     net.Conn
	callMap map[int64]chan callResp
	times   int64
	mux     sync.Mutex
}

func NewClient() *Client {
	return &Client{
		callMap: make(map[int64]chan callResp),
	}
}

func (c *Client) Connect(port string) error {
	c.port = port
	conn, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	c.clt = conn
	go func() {
		if err := c.read(); err != nil {
			log.Printf("Client error: %+v\n", err)
			// c.mux.Lock()
			c.Close()
			// c.mux.Unlock()
			c.mux.Lock()
			for _, cr := range c.callMap {
				cr <- callResp{
					err: fmt.Errorf("client error: %+v", err),
				}
			}
			c.mux.Unlock()

			return
		}
	}()
	return nil
}

func (c *Client) Close() {
	if c.clt != nil {
		c.clt.Close()
	}
}

func (c *Client) Call(server string, fnName string, resp interface{}, params ...interface{}) error {
	if c.clt == nil {
		err := c.Connect(c.port)
		if err != nil {
			return err
		}
	}
	ps := make([]interface{}, 0, len(params))
	for _, p := range params {
		ps = append(ps, p)
	}
	b, err := json.Marshal(ps)
	if err != nil {
		return err
	}
	c.mux.Lock()
	c.times += 1

	h := &header{
		bodyLength: int64(len(b)),
		reqID:      c.times,
		server:     server,
		fnName:     fnName,
	}
	c.mux.Unlock()
	ch := make(chan callResp)
	c.mux.Lock()
	c.callMap[h.reqID] = ch
	c.mux.Unlock()
	err = c.call(c.buildHeader(h), b)
	if err != nil {
		return err
	}

	cr := <-ch
	c.mux.Lock()
	delete(c.callMap, h.reqID)
	c.mux.Unlock()
	if cr.err != nil {
		return cr.err
	}
	if resp == nil {
		return nil
	}
	return json.Unmarshal(cr.resp, resp)
}

type callResp struct {
	err  error
	resp []byte
}

func (c *Client) call(header, body []byte) error {
	req := make([]byte, 0, len(header)+len(body))
	req = append(req, header...)
	req = append(req, body...)
	_, err := c.clt.Write(req)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) read() error {
	for {
		result := make([]byte, headerLength)
		_, err := c.clt.Read(result)
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

		ch, err := c.getCallResp(r.reqID)
		if err != nil {
			return err
		}

		b := make([]byte, r.bodyLength)
		_, err = c.clt.Read(b)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if r.isErr {
			ch <- callResp{
				err: fmt.Errorf("%s", string(b)),
			}
			continue
			// return fmt.Errorf("%s", string(b))
		}

		var v interface{}
		if err = json.Unmarshal(b, &v); err != nil {
			ch <- callResp{
				err: err,
			}
			continue
		}
		value, _ := v.([]interface{})
		if len(value) == 0 {
			ch <- callResp{}
			continue
		}

		rb, err := json.Marshal(value[0])
		if err != nil {
			ch <- callResp{
				err: err,
			}
			continue
		}
		ch <- callResp{
			resp: rb,
		}
	}
}

func (c *Client) getCallResp(key int64) (chan<- callResp, error) {
	c.mux.Lock()
	defer c.mux.Unlock()
	cr, ok := c.callMap[key]
	if !ok {
		return nil, fmt.Errorf("call %d is not exist", key)
	}
	return cr, nil
}

func (c *Client) buildHeader(h *header) []byte {
	h1 := fmt.Sprintf("%02x", h.bodyLength)
	h2 := fmt.Sprintf("%08x", h.reqID)
	sfb := []byte(h.server + "." + h.fnName)
	h3 := fmt.Sprintf("%010x", len(sfb))

	hb := make([]byte, 0, headerLength+len(sfb))
	hb = append(hb, []byte(h1)...)
	hb = append(hb, []byte(h2)...)
	hb = append(hb, []byte(h3)...)
	hb = append(hb, sfb...)
	return hb
}

type result struct {
	isErr      bool
	reqID      int64
	bodyLength int64
}

const IsErr = 1

func (c *Client) resolveResult(b []byte) (*result, error) {
	isErr, err := strconv.ParseInt(string(b[0]), 16, 64)
	if err != nil {
		return nil, err
	}
	reqID, err := strconv.ParseInt(string(b[1:9]), 16, 64)
	if err != nil {
		return nil, err
	}
	length, err := strconv.ParseInt(string(b[9:]), 16, 64)
	if err != nil {
		return nil, err
	}
	r := &result{}
	r.reqID = reqID
	if isErr == IsErr {
		r.isErr = true
	}
	r.bodyLength = length
	// log.Printf("recv rh: %+v\n", r)
	return r, nil
}
