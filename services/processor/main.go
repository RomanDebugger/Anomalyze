package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/segmentio/kafka-go"
)

type MetricMessage struct {
	Hostname  string    `json:"hostname"`
	Timestamp time.Time `json:"timestamp"`
	CPU_Usage float64   `json:"cpu_usage_percent"`
	MEM_Usage float64   `json:"mem_usage_percent"`
}

type InferenceRequest struct {
	Window []MetricMessage `json:"window"`
}

const windowSize = 6

func callInferenceAPI(window []MetricMessage) {
	url := "http://ml-analyzer:8000/infer"

	requestBody := InferenceRequest{Window: window}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Error marshalling inference request: %v", err)
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error calling inference API: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Inference API response status: %s", resp.Status)
}

func main() {
	log.Println("Processor service starting... waiting 10 seconds for services to be ready.")
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

	log.Println("Processor started, listening for messages...")

	var currentWindow []MetricMessage

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		var msg MetricMessage
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Printf("Could not unmarshal message: %v", err)
			continue
		}

		currentWindow = append(currentWindow, msg)

		if len(currentWindow) == windowSize {
			log.Printf("Window is full. Sending to ML service for analysis...")
			callInferenceAPI(currentWindow)

			currentWindow = nil
		}
	}
}
