package main

import (
	"easyrpc"
	"fmt"
	"log"
	"sync"
	"time"
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
	// params := []string{"laowang"}
	t0 := time.Now()
	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		if i < 500 {
			go func() {
				err := c.Call("Hello", "SayHello", nil, "laoliu")
				if err != nil {
					fmt.Println("Call error: ", err)
				}
				wg.Done()
			}()
		} else {
			go func() {
				var resp string
				err := c.Call("Hello", "GetHello", &resp, "laowang")
				if err != nil {
					fmt.Println("Call error: ", err)
				}
				wg.Done()
			}()
		}
		//wg.Done()
	}
	wg.Wait()
	fmt.Printf("concurrent used: %+v\n", time.Since(t0))
	t0 = time.Now()
	for i := 0; i < 500; i++ {
		callSayHello(c)
	}
	for i := 0; i < 500; i++ {
		callGetHello(c)
	}
	fmt.Printf("sync used: %+v\n", time.Since(t0))
	//go func() {
	//	err := c.Call("Hello", "SayHello", nil, "laoliu")
	//	if err != nil {
	//		fmt.Println("Call error: ", err)
	//	}
	//	fmt.Println("Call 1 ok")
	//}()
	//<-time.After(1 * time.Second)
	//var resp string
	//err := c.Call("Hello", "GetHello", &resp, "laowang")
	//if err != nil {
	//	fmt.Println("Call error: ", err)
	//}
	//fmt.Println(resp)
	//fmt.Println("Call 2 ok")
}

func callSayHello(c *easyrpc.Client) {
	err := c.Call("Hello", "SayHello", nil, "laoliu")
	if err != nil {
		fmt.Println("Call error: ", err)
	}
}

func callGetHello(c *easyrpc.Client) {
	var resp string
	err := c.Call("Hello", "GetHello", &resp, "laowang")
	if err != nil {
		fmt.Println("Call error: ", err)
	}
	//fmt.Println(resp)
}
