package mqtt

import (
	"fmt"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	client paho.Client
}

func NewClient(broker, clientID string) *Client {
	opts := paho.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetOnConnectHandler(func(c paho.Client) {
		log.Println("MQTT client connected")
	})
	opts.SetConnectionLostHandler(func(c paho.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	})
	return &Client{
		client: paho.NewClient(opts),
	}
}

func (c *Client) Connect() {
	token := c.client.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Printf("MQTT connect error: %v", token.Error())
	}
}

func (c *Client) Publish(topic string, qos byte, payload interface{}) {
	token := c.client.Publish(topic, qos, false, payload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("MQTT publish error to %s: %v", topic, token.Error())
	} else {
		log.Printf("MQTT published to %s: %v", topic, payload)
	}
}

func (c *Client) Subscribe(topic string, qos byte, callback paho.MessageHandler) {
	token := c.client.Subscribe(topic, qos, callback)
	token.Wait()
	if token.Error() != nil {
		log.Printf("MQTT subscribe error to %s: %v", topic, token.Error())
	} else {
		log.Printf("MQTT subscribed to %s", topic)
	}
}

func (c *Client) PublishValveControl(partitionID string, action string) {
	topic := fmt.Sprintf("control/valve/%s", partitionID)
	c.Publish(topic, 1, action)
}

func (c *Client) PublishFanControl(partitionID string, action string) {
	topic := fmt.Sprintf("control/fan/%s", partitionID)
	c.Publish(topic, 1, action)
}

func (c *Client) PublishAlarm(level int, message string) {
	topic := fmt.Sprintf("alarm/%d", level)
	c.Publish(topic, 1, message)
}

func (c *Client) PublishEvacuation(partitionID string) {
	c.Publish("notification/evacuation", 1, fmt.Sprintf("Evacuate partition %s immediately", partitionID))
}

func (c *Client) Disconnect() {
	c.client.Disconnect(250)
	log.Println("MQTT client disconnected")
}
