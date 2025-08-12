package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type MetricMessage struct {
	Hostname  string    `json:"hostname"`
	Timestamp time.Time `json:"timestamp"`
	CPU_Usage float64   `json:"cpu_usage_percent"`
	MEM_Usage float64   `json:"mem_usage_percent"`
}

const (
	windowSize = 6
)

func main() {
	time.Sleep(10 * time.Second)
	kafkaBroker := "kafka:29092"
	topic := "metrics-raw"
	groupID := "anomaly-processor-group"

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer r.Close()

	log.Println("Processor started, listening for messages on topic:", topic)
	var currentWindow []MetricMessage

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error while reading message: %v", err)
			break
		}

		var msg MetricMessage
		err = json.Unmarshal(m.Value, &msg)
		if err != nil {
			log.Printf("Could not unmarshal message: %v", err)
			continue
		}
		log.Printf("Parsed message for host %s: CPU=%.2f%%, MEM=%.2f%%", msg.Hostname, msg.CPU_Usage, msg.MEM_Usage)
		currentWindow = append(currentWindow, msg)
		if len(currentWindow) == windowSize {
			log.Printf(">>>> Window is full with %d messages! Ready for analysis.", windowSize)
			for _, metric := range currentWindow {
				log.Printf("  - Host: %s, CPU: %.2f%%", metric.Hostname, metric.CPU_Usage)
			}
			currentWindow = nil
			log.Println(">>>> Window cleared.")
		}
	}
}
