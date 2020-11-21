package easyrpc

import (
	"easyrpc/rreflect"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"strconv"
	"strings"
)

//type Server interface {
//	Regist(name string, server interface{}) error
//	Call(server string, fnName string, params ...interface{}) (interface{}, error)
//	StartServer(port string) error
//}

func NewServer(port string) *Server {
	return &Server{
		port:    port,
		servers: make(map[string]interface{}),
		funcs:   make(map[string]reflect.Value),
	}
}

type Server struct {
	port    string
	servers map[string]interface{}
	funcs   map[string]reflect.Value
	l       net.Listener
}

func (r *Server) Rigist(name string, server interface{}) error {
	num, funcs := rreflect.GetAllFn(server)
	if num == 0 {
		return fmt.Errorf("rigist error: the server have no func")
	}
	r.servers[name] = server
	for k, v := range funcs {
		r.funcs[r.buildFuncName(name, k)] = v
	}
	return nil
}

func (r *Server) Call(server string, fnName string, params ...interface{}) (interface{}, error) {
	s, ok := r.servers[server]
	if !ok {
		return nil, fmt.Errorf("rpc error: server %s is not exist", server)
	}
	fv, ok := r.funcs[r.buildFuncName(server, fnName)]
	if !ok {
		return nil, fmt.Errorf("rpc error: func %s is not exist", fnName)
	}
	ps := make([]interface{}, 0, len(params))
	ps = append(ps, s)
	ps = append(ps, params...)

	vs, err := rreflect.Call(fv, ps)
	if err != nil {
		return nil, err
	}
	return vs, nil
}

func (r *Server) StartServer() error {
	if r.port == "" {
		r.port = ":23333"
	}
	l, err := net.Listen("tcp", r.port)
	if err != nil {
		return err
	}
	log.Printf("start rpc server on %s\n", r.port)
	r.l = l

	for {
		conn, err := r.l.Accept()
		if err != nil {
			log.Printf("accpet error: %s\n", err)
			continue
		}
		log.Printf("connect with %s\n", conn.RemoteAddr().String())

		go r.serve(conn)
	}
}

const headerLength = 8

func (r *Server) serve(conn net.Conn) {
	// TODO Multiplexing
	defer func() {
		conn.Close()
		log.Printf("connect with %s closed\n", conn.RemoteAddr().String())
		if err := recover(); err != nil {
			log.Printf("deal conn error: %s", err)
			return
		}
	}()

	for {
		header := make([]byte, headerLength)
		_, err := conn.Read(header)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("rpc error: %s\n", err)
			r.dealErr(conn, fmt.Sprintf("rpc error: %s", err))
			return
		}
		h, err := r.resolveHeader(header)
		if err != nil {
			log.Printf("rpc error: %s\n", err)
			r.dealErr(conn, fmt.Sprintf("rpc error: %s", err))
			return
		}
		head := make([]byte, h.headLength)
		_, err = conn.Read(head)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Printf("rpc error: %s\n", err)
			r.dealErr(conn, fmt.Sprintf("rpc error: %s", err))
			return
		}
		strs := strings.Split(string(head), ".")
		if len(strs) != 2 {
			log.Printf("rpc error: resolveHeader error: bad request\n")
			r.dealErr(conn, "rpc error: resolveHeader error: bad request")
			return
		}
		h.server = strs[0]
		h.fnName = strs[1]

		body := make([]byte, h.bodyLength)
		_, err = conn.Read(body)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Printf("rpc error: %s\n", err)
			r.dealErr(conn, fmt.Sprintf("rpc error: %s", err))
			return
		}
		params := make([]interface{}, 0)
		if len(body) != 0 {
			if err := json.Unmarshal(body, &params); err != nil {
				log.Printf("rpc error: %s\n", err)
				r.dealErr(conn, fmt.Sprintf("rpc error: %s", err))
				return
			}
		}
		v, err := r.Call(h.server, h.fnName, params...)
		if err != nil {
			log.Printf("rpc error: %s\n", err)
			r.dealErr(conn, fmt.Sprintf("rpc error: %s", err))
			// 业务调用失败，可以继续
			continue
		}
		r.dealResp(conn, v)
	}

}

type header struct {
	bodyLength int64
	headLength int64
	server     string
	fnName     string
}

func (r *Server) resolveHeader(hb []byte) (*header, error) {
	length, err := strconv.ParseInt(string(hb[:2]), 16, 64)
	if err != nil {
		return nil, err
	}
	h := &header{}
	h.bodyLength = length

	length, err = strconv.ParseInt(string(hb[2:]), 16, 64)
	if err != nil {
		return nil, err
	}
	h.headLength = length

	return h, nil
}

func (r *Server) buildFuncName(server, fnName string) string {
	return server + "." + fnName
}

func (r *Server) dealErr(conn net.Conn, errMsg string) {
	r1 := fmt.Sprintf("%x", 1)
	r3 := []byte(errMsg)
	r2 := fmt.Sprintf("%07x", len(r3))

	b := make([]byte, 0, headerLength+len(r3))
	b = append(b, []byte(r1)...)
	b = append(b, []byte(r2)...)
	b = append(b, r3...)

	_, err := conn.Write(b)
	if err != nil {
		log.Printf("rpc err: %s", err)
	}
}
func (r *Server) dealResp(conn net.Conn, v interface{}) {
	r1 := fmt.Sprintf("%x", 0)

	resp, err := json.Marshal(v)
	if err != nil {
		log.Printf("rpc error: %s\n", err)
		r.dealErr(conn, fmt.Sprintf("rpc error: %s", err))
		return
	}
	r2 := fmt.Sprintf("%07x", len(resp))

	b := make([]byte, 0, headerLength+len(resp))
	b = append(b, []byte(r1)...)
	b = append(b, []byte(r2)...)
	b = append(b, resp...)

	_, err = conn.Write(b)
	if err != nil {
		log.Printf("rpc err: %s", err)
	}
}
