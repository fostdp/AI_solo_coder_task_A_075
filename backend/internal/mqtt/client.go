package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type CommandStatus string

const (
	CommandPending   CommandStatus = "pending"
	CommandAcked     CommandStatus = "acked"
	CommandTimeout   CommandStatus = "timeout"
	CommandFailed    CommandStatus = "failed"
)

type PendingCommand struct {
	ID          string        `json:"id"`
	Topic       string        `json:"topic"`
	Payload     string        `json:"payload"`
	QoS         byte          `json:"qos"`
	PartitionID string        `json:"partition_id"`
	DeviceType  string        `json:"device_type"`
	Action      string        `json:"action"`
	Status      CommandStatus `json:"status"`
	Retries     int           `json:"retries"`
	MaxRetries  int           `json:"max_retries"`
	CreatedAt   time.Time     `json:"created_at"`
	LastSentAt  time.Time     `json:"last_sent_at"`
	AckTimeout  time.Duration `json:"ack_timeout"`
}

type CommandResult struct {
	CommandID   string        `json:"command_id"`
	PartitionID string        `json:"partition_id"`
	DeviceType  string        `json:"device_type"`
	Action      string        `json:"action"`
	Status      CommandStatus `json:"status"`
	Retries     int           `json:"retries"`
	Timestamp   time.Time     `json:"timestamp"`
}

type Client struct {
	client        paho.Client
	pending       map[string]*PendingCommand
	pendingMu     sync.RWMutex
	resultChan    chan CommandResult
	ackTimeout    time.Duration
	maxRetries    int
	retryInterval time.Duration
	quit          chan struct{}
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

	c := &Client{
		client:        paho.NewClient(opts),
		pending:       make(map[string]*PendingCommand),
		resultChan:    make(chan CommandResult, 100),
		ackTimeout:    5 * time.Second,
		maxRetries:    3,
		retryInterval: 2 * time.Second,
		quit:          make(chan struct{}),
	}

	return c
}

func (c *Client) Connect() {
	token := c.client.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Printf("MQTT connect error: %v", token.Error())
	}

	c.subscribeAckTopics()
	go c.retryLoop()
}

func (c *Client) subscribeAckTopics() {
	ackTopics := []string{
		"control/valve/+/ack",
		"control/fan/+/ack",
	}
	for _, topic := range ackTopics {
		c.Subscribe(topic, 1, func(client paho.Client, msg paho.Message) {
			c.handleAck(msg.Topic(), msg.Payload())
		})
	}
}

func (c *Client) handleAck(topic string, payload []byte) {
	var ack struct {
		CommandID string `json:"command_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(payload, &ack); err != nil {
		log.Printf("Failed to parse ack message: %v", err)
		return
	}

	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	cmd, exists := c.pending[ack.CommandID]
	if !exists {
		return
	}

	if ack.Status == "ok" {
		cmd.Status = CommandAcked
		result := CommandResult{
			CommandID:   cmd.ID,
			PartitionID: cmd.PartitionID,
			DeviceType:  cmd.DeviceType,
			Action:      cmd.Action,
			Status:      CommandAcked,
			Retries:     cmd.Retries,
			Timestamp:   time.Now(),
		}
		delete(c.pending, cmd.ID)
		select {
		case c.resultChan <- result:
		default:
		}
	}
}

func (c *Client) Results() <-chan CommandResult {
	return c.resultChan
}

func (c *Client) retryLoop() {
	ticker := time.NewTicker(c.retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkAndRetry()
		case <-c.quit:
			return
		}
	}
}

func (c *Client) checkAndRetry() {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	now := time.Now()
	var timedOut []string

	for id, cmd := range c.pending {
		if cmd.Status != CommandPending {
			continue
		}

		if now.Sub(cmd.LastSentAt) > cmd.AckTimeout {
			if cmd.Retries >= cmd.MaxRetries {
				cmd.Status = CommandTimeout
				timedOut = append(timedOut, id)
				result := CommandResult{
					CommandID:   cmd.ID,
					PartitionID: cmd.PartitionID,
					DeviceType:  cmd.DeviceType,
					Action:      cmd.Action,
					Status:      CommandTimeout,
					Retries:     cmd.Retries,
					Timestamp:   now,
				}
				select {
				case c.resultChan <- result:
				default:
				}
				log.Printf("[MQTT] Command %s timed out after %d retries (device: %s/%s)",
					id, cmd.MaxRetries, cmd.DeviceType, cmd.PartitionID)
			} else {
				cmd.Retries++
				cmd.LastSentAt = now
				token := c.client.Publish(cmd.Topic, cmd.QoS, false, cmd.Payload)
				token.Wait()
				if token.Error() != nil {
					log.Printf("[MQTT] Retry %d failed for command %s: %v", cmd.Retries, id, token.Error())
				} else {
					log.Printf("[MQTT] Retry %d sent for command %s to %s", cmd.Retries, id, cmd.Topic)
				}
			}
		}
	}

	for _, id := range timedOut {
		delete(c.pending, id)
	}
}

func (c *Client) publishWithAck(topic string, qos byte, partitionID, deviceType, action string, payload interface{}) string {
	cmdID := fmt.Sprintf("cmd-%d", time.Now().UnixNano())

	payloadBytes, err := json.Marshal(map[string]interface{}{
		"command_id":   cmdID,
		"partition_id": partitionID,
		"device_type":  deviceType,
		"action":       action,
		"data":         payload,
	})
	if err != nil {
		log.Printf("[MQTT] Failed to marshal command payload: %v", err)
		return cmdID
	}

	cmd := &PendingCommand{
		ID:          cmdID,
		Topic:       topic,
		Payload:     string(payloadBytes),
		QoS:         qos,
		PartitionID: partitionID,
		DeviceType:  deviceType,
		Action:      action,
		Status:      CommandPending,
		Retries:     0,
		MaxRetries:  c.maxRetries,
		CreatedAt:   time.Now(),
		LastSentAt:  time.Now(),
		AckTimeout:  c.ackTimeout,
	}

	c.pendingMu.Lock()
	c.pending[cmdID] = cmd
	c.pendingMu.Unlock()

	token := c.client.Publish(topic, qos, false, string(payloadBytes))
	token.Wait()
	if token.Error() != nil {
		log.Printf("[MQTT] Publish error to %s: %v", topic, token.Error())
	} else {
		log.Printf("[MQTT] Published to %s: command_id=%s", topic, cmdID)
	}

	return cmdID
}

func (c *Client) Publish(topic string, qos byte, payload interface{}) {
	token := c.client.Publish(topic, qos, false, payload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("[MQTT] publish error to %s: %v", topic, token.Error())
	}
}

func (c *Client) Subscribe(topic string, qos byte, callback paho.MessageHandler) {
	token := c.client.Subscribe(topic, qos, callback)
	token.Wait()
	if token.Error() != nil {
		log.Printf("[MQTT] subscribe error to %s: %v", topic, token.Error())
	}
}

func (c *Client) PublishValveControl(partitionID string, action string) string {
	topic := fmt.Sprintf("control/valve/%s", partitionID)
	return c.publishWithAck(topic, 1, partitionID, "valve", action, action)
}

func (c *Client) PublishFanControl(partitionID string, action string) string {
	topic := fmt.Sprintf("control/fan/%s", partitionID)
	return c.publishWithAck(topic, 1, partitionID, "fan", action, action)
}

func (c *Client) PublishAlarm(level int, message string) {
	topic := fmt.Sprintf("alarm/%d", level)
	c.Publish(topic, 1, message)
}

func (c *Client) PublishEvacuation(partitionID string) {
	c.Publish("notification/evacuation", 1, fmt.Sprintf("Evacuate partition %s immediately", partitionID))
}

func (c *Client) GetPendingCount() int {
	c.pendingMu.RLock()
	defer c.pendingMu.RUnlock()
	return len(c.pending)
}

func (c *Client) Disconnect() {
	close(c.quit)
	c.client.Disconnect(250)
	log.Println("MQTT client disconnected")
}

func (c *Client) Client() paho.Client {
	return c.client
}

func (c *Client) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
