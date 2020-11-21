package main

import (
	"easyrpc"
	"fmt"
	"log"
)

func main() {
	//for i := 0; i < 10; i++ {
	//	go hello()
	//}
	//<-time.After(10 * time.Second)
	hello()
}

func hello() {
	c := easyrpc.NewClient()
	if err := c.Connect(":23333"); err != nil {
		log.Fatal(err)
	}
	//params := []string{"laowang"}
	err := c.Call("Hello", "SayHello", nil, 1)
	if err != nil {
		fmt.Println("Call error: ", err)
	}
	var resp string
	err = c.Call("Hello", "GetHello", &resp, "laowang")
	if err != nil {
		fmt.Println("Call error: ", err)
	}
	fmt.Println(resp)
	err = c.Call("Hello", "NoHello", &resp, "laowang")
	if err != nil {
		fmt.Println("Call error: ", err)
	}
	err = c.Call("Hello", "GetHello", &resp, "laoliu")
	if err != nil {
		fmt.Println("Call error: ", err)
	}
	fmt.Println(resp)
}
