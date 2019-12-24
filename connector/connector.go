//Package connector .
package connector

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/streadway/amqp"
)

//Connection connection wrapper
type Connection struct {
	*amqp.Connection
	notifyClose chan *amqp.Error
	delay       time.Duration
	log         Logger
	url         string
}

//SetLog .
func (c *Connection) SetLog(log Logger) {
	c.log = log
}

//SetDelay .
func (c *Connection) SetDelay(delay time.Duration) {
	c.delay = delay
}

//Channel channel wrapper
type Channel struct {
	*amqp.Channel
	notifyClose chan *amqp.Error
	delay       time.Duration
	connection  *Connection
	closed      int32
}

//SetDelay .
func (c *Channel) SetDelay(delay time.Duration) {
	c.delay = delay
}

// Close close channeland flag .
func (c *Channel) Close() error {
	if c.IsClosed() {
		return amqp.ErrClosed
	}

	atomic.StoreInt32(&c.closed, int32(1))

	return c.Channel.Close()
}

// IsClosed check channel is closed by user, not network error.
func (c *Channel) IsClosed() bool {
	if atomic.LoadInt32(&c.closed) != 0 {
		return true
	}
	return false
}

// Logger .
type Logger interface {
	Log(format string, a ...interface{})
}

type defaultLogger struct {
}

func (d defaultLogger) Log(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

//Dial accepts a string in the AMQP URI format and returns a new Connection
func Dial(url string) (*Connection, error) {
	conn, err := dial(url)
	if err != nil {
		return nil, err
	}

	connection := &Connection{
		Connection: conn,
		url:        url,
		delay:      5 * time.Second,
		log:        defaultLogger{},
	}

	go connection.reConnector()
	connection.log.Log("conn rabbit connected .. \n")
	return connection, nil
}

func dial(url string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *Connection) reConnector() {
	for {
		c.notifyClose = c.NotifyClose(make(chan *amqp.Error))
		select {
		case amqpErr, ok := <-c.notifyClose:
			if conn, err := dial(c.url); err != nil {
				c.log.Log("connection reConnector rabbit connect error : %v, channel status: %t \n", amqpErr, ok)
				time.Sleep(c.delay)
				c.notifyClose = make(chan *amqp.Error)
				go send2ConnCloseErr(c.notifyClose, err)
			} else {
				c.Connection = conn
				c.log.Log("connection reConnector rabbit reconnected .. \n")
			}
		}
	}
}

//Channel get reconnect channel .
func (c *Connection) Channel() (*Channel, error) {
	ch, err := c.Connection.Channel()
	if err != nil {
		return nil, err
	}

	channel := &Channel{
		Channel:    ch,
		delay:      c.delay,
		connection: c,
	}
	go channel.reConnector(c)
	return channel, nil
}

func (c *Channel) reConnector(connection *Connection) {
	for {
		c.notifyClose = c.NotifyClose(make(chan *amqp.Error))
		select {
		case amqpErr, ok := <-c.notifyClose:
			if ch, err := connection.Connection.Channel(); err != nil {
				connection.log.Log("channel reConnector rabbit connect error : %v, channel status: %t \n", amqpErr, ok)
				time.Sleep(c.delay)
				c.notifyClose = make(chan *amqp.Error)
				go send2ConnCloseErr(c.notifyClose, err)
			} else {
				c.Channel = ch
				connection.log.Log("channel reConnector rabbit channel reconnected .. \n")
			}
		}
	}
}

//Consume .
func (c *Channel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	deliveryCh := make(chan amqp.Delivery)
	go func() {
		for {
			msgs, err := c.Channel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
			if err != nil {
				c.connection.log.Log("consume consume err: %s \n", err)
				time.Sleep(c.delay)
				continue
			}

			for msg := range msgs {
				deliveryCh <- msg
			}

			time.Sleep(c.delay)

			if c.IsClosed() {
				break
			}
		}
	}()
	return deliveryCh, nil
}

func send2ConnCloseErr(notifyClose chan *amqp.Error, err error) {
	notifyClose <- &amqp.Error{
		Reason: err.Error(),
	}
}
