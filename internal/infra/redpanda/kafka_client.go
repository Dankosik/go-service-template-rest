package redpanda

import (
	"context"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type ClientConfig struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
}

type BrokerProbe struct {
	brokers []string
	timeout time.Duration
}

func NewBrokerProbe(brokers []string, timeout time.Duration) BrokerProbe {
	return BrokerProbe{
		brokers: cleanBrokers(brokers),
		timeout: timeout,
	}
}

func (p BrokerProbe) Name() string {
	return "redpanda"
}

func (p BrokerProbe) Check(ctx context.Context) error {
	brokers := cleanBrokers(p.brokers)
	if len(brokers) == 0 {
		return fmt.Errorf("%w: broker list is required", ErrInvalidEvent)
	}
	timeout := p.timeout
	if timeout <= 0 {
		timeout = time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("%w: broker probe: %w", ErrRetryable, err)
	}
	_ = conn.Close()
	return nil
}

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(cfg ClientConfig) (*KafkaConsumer, error) {
	brokers := cleanBrokers(cfg.Brokers)
	if len(brokers) == 0 || strings.TrimSpace(cfg.Topic) == "" || strings.TrimSpace(cfg.ConsumerGroup) == "" {
		return nil, fmt.Errorf("%w: brokers, topic, and consumer group are required", ErrInvalidEvent)
	}
	return &KafkaConsumer{reader: kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    strings.TrimSpace(cfg.Topic),
		GroupID:  strings.TrimSpace(cfg.ConsumerGroup),
		MinBytes: 1,
		MaxBytes: 10e6,
	})}, nil
}

func (c *KafkaConsumer) FetchMessage(ctx context.Context) (Message, error) {
	if c == nil || c.reader == nil {
		return Message{}, fmt.Errorf("%w: consumer is not configured", ErrInvalidEvent)
	}
	msg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return Message{}, fmt.Errorf("fetch kafka message: %w", err)
	}
	headers := make(map[string]string, len(msg.Headers))
	for _, header := range msg.Headers {
		headers[header.Key] = string(header.Value)
	}
	if msg.Partition < 0 || msg.Partition > math.MaxInt32 {
		return Message{}, fmt.Errorf("%w: kafka partition is outside int32 range", ErrInvalidEvent)
	}
	return Message{
		Topic:     msg.Topic,
		Partition: int32(msg.Partition),
		Offset:    msg.Offset,
		Key:       append([]byte(nil), msg.Key...),
		Value:     append([]byte(nil), msg.Value...),
		Headers:   headers,
	}, nil
}

func (c *KafkaConsumer) CommitOffset(ctx context.Context, msg Message) error {
	if c == nil || c.reader == nil {
		return fmt.Errorf("%w: consumer is not configured", ErrInvalidEvent)
	}
	if err := c.reader.CommitMessages(ctx, kafka.Message{
		Topic:     msg.Topic,
		Partition: int(msg.Partition),
		Offset:    msg.Offset,
	}); err != nil {
		return fmt.Errorf("commit kafka offset: %w", err)
	}
	return nil
}

func (c *KafkaConsumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("close kafka consumer: %w", err)
	}
	return nil
}

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string) (*KafkaProducer, error) {
	clean := cleanBrokers(brokers)
	if len(clean) == 0 {
		return nil, fmt.Errorf("%w: broker list is required", ErrInvalidEvent)
	}
	return &KafkaProducer{writer: &kafka.Writer{
		Addr:         kafka.TCP(clean...),
		RequiredAcks: kafka.RequireAll,
		Balancer:     &kafka.Hash{},
		Async:        false,
	}}, nil
}

func (p *KafkaProducer) Produce(ctx context.Context, msg ProduceMessage) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("%w: producer is not configured", ErrInvalidEvent)
	}
	headers := make([]kafka.Header, 0, len(msg.Headers))
	for key, value := range msg.Headers {
		headers = append(headers, kafka.Header{Key: key, Value: []byte(value)})
	}
	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic:   msg.Topic,
		Key:     append([]byte(nil), msg.Key...),
		Value:   append([]byte(nil), msg.Value...),
		Headers: headers,
	}); err != nil {
		return fmt.Errorf("produce kafka message: %w", err)
	}
	return nil
}

func (p *KafkaProducer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("close kafka producer: %w", err)
	}
	return nil
}

func cleanBrokers(brokers []string) []string {
	clean := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		if trimmed := strings.TrimSpace(broker); trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	return clean
}
