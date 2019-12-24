package connector

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/streadway/amqp"
)

var (
	TestURL          = "amqp://guest:guest@localhost:5672/"
	TestExchangeName = "amqp-test-reconnect-exchange-name"
	TestQueueName    = "amqp-test-reconnect-queue-name"
)

func TestDial(t *testing.T) {

	conn, err := Dial(TestURL)
	if err != nil {
		t.Fatal(err)
		return
	}

	{ //查看链接是否建立
		if conn.IsClosed() {
			t.Fatal(err)
			return
		}
	}

	//手动关闭链接
	err = conn.Close()
	if err != nil {
		t.Fatal(err)
		return
	}

	{
		if conn.IsClosed() { //如果链接关闭， 则正常, 因为重连需要更多的时间
			t.Log("connection is closed")
		} else {
			t.Fatal("connection should closed")
		}

	}

	time.Sleep(1 * time.Second)
	{ //等待2秒钟后， 如果还没有连上则代表没有重启
		if conn.IsClosed() {
			t.Fatal(err)
			return
		}
	}
}

func TestChannel(t *testing.T) {
	conn, err := Dial(TestURL)
	if err != nil {
		t.Fatal(err)
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
		return
	}

	//声明一个测试队列
	err = ch.ExchangeDeclare(TestExchangeName, amqp.ExchangeTopic, true, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
		return
	}

	//声明一个测试队列
	_, err = ch.QueueDeclare(TestQueueName, true, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
		return
	}

	//将交换机跟队列进行绑定
	if err := ch.QueueBind(TestQueueName, "", TestExchangeName, false, nil); err != nil {
		t.Fatal(err)
		return
	}

	//声明跟绑定如果有则直接返回成功， 如果没有则声明或者绑定

	//测试publish
	{
		err := ch.Publish(TestExchangeName, "", false, false, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(time.Now().String()),
		})
		if err != nil {
			t.Fatal(err)
			return
		}
	}

	//重启conn
	conn.Close()

	//测试publish, 刚重启过， 现在还没有重启成功
	{
		err := ch.Publish(TestExchangeName, "", false, false, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(time.Now().String()),
		})
		if err == nil {
			t.Fatal("current restart server , now should not init success .")
			return
		}
	}

	//时间要久一点， 因为要先启动tcp， 然后在启动通道
	time.Sleep(10 * time.Second)

	//测试publish, 刚重启过， 重启成功了
	{
		err := ch.Publish(TestExchangeName, "", false, false, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(time.Now().String()),
		})
		if err != nil {
			t.Fatal(err)
			return
		}
	}
}

func TestConsumeNormal(t *testing.T) {
	wg := sync.WaitGroup{}
	conn, err := Dial(TestURL)
	if err != nil {
		t.Fatal(err)
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
		return
	}

	consumeCh, err := ch.Consume(TestQueueName, "", true, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
		return
	}

	{ //测试consume 能否接收到信息
		var sendStr = "test"
		wg.Add(2)
		go func() {
			for msg := range consumeCh {
				receiveStr := string(msg.Body)
				if sendStr == receiveStr {
					t.Log("receive need data")
					wg.Done()
				}
			}
		}()

		go func() {
			time.Sleep(10 * time.Second)
			defer wg.Done()
			err := ch.Publish(TestExchangeName, "", false, false, amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(sendStr),
			})
			if err != nil {
				t.Fatal(err)
				return
			}
		}()
		wg.Wait()
	}
}

func TestConsumeAbnormal(t *testing.T) {
	wg := sync.WaitGroup{}
	conn, err := Dial(TestURL)
	if err != nil {
		t.Fatal(err)
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
		return
	}

	consumeCh, err := ch.Consume(TestQueueName, "", true, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
		return
	}

	{ //测试consume 能否接收到信息
		var sendStr = "test2"
		wg.Add(2)
		go func() {
			for msg := range consumeCh {
				fmt.Println(msg)
				fmt.Println(string(msg.Body))
				receiveStr := string(msg.Body)
				if sendStr == receiveStr {
					t.Log("receive need data")
					wg.Done()
				}
			}
		}()

		err = ch.Channel.Close()
		if err != nil {
			t.Error(err)
		}

		go func() {
			time.Sleep(10 * time.Second)
			defer wg.Done()
			err := ch.Publish(TestExchangeName, "", false, false, amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(sendStr),
			})
			if err != nil {
				t.Fatal(err)
				return
			}

		}()
		wg.Wait()
	}
}
