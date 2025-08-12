package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
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

type InferenceResponse struct {
	Status       string  `json:"status"`
	AnomalyScore float64 `json:"anomaly_score"`
	IsAnomaly    bool    `json:"is_anomaly"`
	ModelVersion string  `json:"model_version"`
}

const windowSize = 6

var db *sql.DB

func initDB() {
	connStr := "postgres://user:password@postgres:5432/anomalydb?sslmode=disable"
	var err error
	for i := 0; i < 5; i++ {
		db, err = sql.Open("pgx", connStr)
		if err == nil {
			if err = db.Ping(); err == nil {
				log.Println("Successfully connected to the database!")
				return
			}
		}
		log.Printf("Could not connect to database. Retrying in 5 seconds... (%d/5)", i+1)
		time.Sleep(5 * time.Second)
	}
	log.Fatalf("Could not connect to the database after several retries: %v", err)
}

func createAnomaliesTable() {
	createTableSQL := `CREATE TABLE IF NOT EXISTS anomalies (
		id SERIAL PRIMARY KEY,
		hostname TEXT,
		metric TEXT,
		window_start TIMESTAMP,
		window_end TIMESTAMP,
		anomaly_score REAL,
		model_version TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Could not create anomalies table: %v", err)
	}
	log.Println("Anomalies table is ready.")
}

func insertAnomaly(window []MetricMessage, score float64, version string) {
	sqlStatement := `
		INSERT INTO anomalies (hostname, metric, window_start, window_end, anomaly_score, model_version)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := db.Exec(sqlStatement, window[0].Hostname, "cpu_usage", window[0].Timestamp, window[len(window)-1].Timestamp, score, version)
	if err != nil {
		log.Printf("Failed to insert anomaly into database: %v", err)
	} else {
		log.Println("Successfully inserted anomaly into database.")
	}
}

func callInferenceAPI(window []MetricMessage) {
	url := "http://ml-analyzer:8000/infer"
	requestBody := InferenceRequest{Window: window}
	payload, _ := json.Marshal(requestBody)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error calling inference API: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var inferenceResp InferenceResponse
	if err := json.Unmarshal(body, &inferenceResp); err != nil {
		log.Printf("Could not parse API response: %v", err)
		return
	}

	log.Printf("Inference result: Score=%.2f, IsAnomaly=%t", inferenceResp.AnomalyScore, inferenceResp.IsAnomaly)
	if inferenceResp.IsAnomaly {
		log.Println("Anomaly detected! Storing in database...")
		insertAnomaly(window, inferenceResp.AnomalyScore, inferenceResp.ModelVersion)
	}
}

func main() {
	log.Println("Processor service starting...")

	initDB()
	createAnomaliesTable()

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
	log.Println("Processor connected to Kafka, listening for messages...")

	var currentWindow []MetricMessage
	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}

		var msg MetricMessage
		if err := json.Unmarshal(m.Value, &msg); err != nil {
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
