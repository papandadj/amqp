# AMQP RabbitMQ Reconnect

This library is AMQP library add reconnect function.

## About The Project
 
 RabbitMQ is based on trusted message queues , Therefore , some libraries do not implement retry function.  But in real project, we need a socket that's always connected to receive data or send data. This project is designed to simplify retry operation.

 
## Usage

This is some examples of how to use on your project . Before start, you can test the project.

#### Import

```go get github.com/papandadj/amqp```

#### Example

receive.go

```golang
package main

import (
	"fmt"

	"github.com/papandadj/amqp/connector"
)

var (
	testExchangeName = "amqp-test-reconnect-exchange-name"
	testQueueName    = "amqp-test-reconnect-queue-name"
)

func main() {
	//conn will reconnect if socket disconnect
	conn, err := connector.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		panic(err)
	}

	//create channel auto autoreconnect
	ch, err := conn.Channel()
	if err != nil {
		fmt.Println(err)
		return
	}

	//declare exchange
	err = ch.ExchangeDeclare(testExchangeName, "topic", true, false, false, false, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	//declare queue
	_, err = ch.QueueDeclare(testQueueName, true, false, false, false, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	//queue is bound to exchange
	if err := ch.QueueBind(testQueueName, "", testExchangeName, false, nil); err != nil {
		fmt.Println(err)
		return
	}

	//create consume never down unless the developer themeselves close the channel.
	consumeCh, err := ch.Consume(testQueueName, "", true, false, false, false, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	for msg := range consumeCh {
		//receive data from rabbit
		fmt.Println(msg)
	}
}

```


send.go

```golang
package main

import (
	"fmt"
	"time"

	"github.com/streadway/amqp"

	"github.com/papandadj/amqp/connector"
)

var (
	testExchangeName = "amqp-test-reconnect-exchange-name"
	testQueueName    = "amqp-test-reconnect-queue-name"
)

func main() {
	//conn will reconnect if socket disconnect
	conn, err := connector.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		panic(err)
	}

	//create channel auto autoreconnect
	ch, err := conn.Channel()
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		err := ch.Publish(testExchangeName, "", false, false, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(time.Now().String()),
		})
		if err != nil {
			fmt.Println(err)
		}

		time.Sleep(5 * time.Second)
	}
}
```

Now, ensure that the rabbit service is enabled.

Open terminal, type `go run receive.go`.  Reopen new terminal , type `go run send.go`. We can see that the receiver is always receiving information from rabbit.  If stop rabbit service, `systemctl stop rabbitmq-server.service`, receiver will not receive the message. Finally, open rabbit service, `systemctl restart rabbitmq-server.service` , the receive will auto get message from rabbit, and sender can messages to rabbit service.

#### ReSet Default Configure

```golang 
//reset connection retry time
conn.SetDelay(10 * time.Second)
//reset log
conn.SetLog()
//reset channel and consume retry time
conn.SetDelay(10 * time.Second)
```

