package main

import (
	"amqp/connector"
	"fmt"
	"time"
)

func main() {
	conn, err := connector.Dial("amqp://guest:guest@localhost:5672/")

	if err != nil {
		panic(err)
	}
	fmt.Println(conn)

	ch, err := conn.Channel()
	if err != nil {
		fmt.Println("errrrrrrr", err)
	}

	time.Sleep(3 * time.Second)
	fmt.Println(ch.Close())

	time.Sleep(3 * time.Second)
	fmt.Println(ch.Close())

	time.Sleep(3 * time.Second)
	fmt.Println(ch.Close())

	time.Sleep(1 * time.Hour)
}
