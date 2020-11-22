package main

import (
	"easyrpc"
	"log"
	"time"
)

type Hello struct{}

func (h *Hello) SayHello(name string) {
	<-time.After(500 * time.Millisecond)
	log.Println("Hello ", name)
}

func (h *Hello) GetHello(name string) string {
	// <-time.After(5 * time.Second)
	return "Hello " + name
}

func main() {
	r := easyrpc.NewServer(":23333")
	if err := r.Rigist("Hello", new(Hello)); err != nil {
		log.Fatal(err)
	}
	if err := r.StartServer(); err != nil {
		log.Fatal(err)
	}
}
