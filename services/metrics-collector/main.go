package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	cpuUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "node_cpu_usage_percent",
		Help: "Current CPU usage percentage.",
	})
	memUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "node_memory_usage_percent",
		Help: "Current memory usage percentage.",
	})
)

type MetricMessage struct {
	Hostname  string    `json:"hostname"`
	Timestamp time.Time `json:"timestamp"`
	CPU_Usage float64   `json:"cpu_usage_percent"`
	MEM_Usage float64   `json:"mem_usage_percent"`
}

func recordMetrics(kafkaWriter *kafka.Writer) {
	hostname, _ := os.Hostname()

	go func() {
		for {
			cpuPercentages, err := cpu.Percent(time.Second, false)
			if err != nil {
				log.Printf("Error getting CPU usage: %v", err)
				continue
			}
			cpuVal := cpuPercentages[0]

			vMem, err := mem.VirtualMemory()
			if err != nil {
				log.Printf("Error getting memory usage: %v", err)
				continue
			}
			memVal := vMem.UsedPercent
			cpuUsage.Set(cpuVal)
			memUsage.Set(memVal)

			msg := MetricMessage{
				Hostname:  hostname,
				Timestamp: time.Now(),
				CPU_Usage: cpuVal,
				MEM_Usage: memVal,
			}

			jsonMsg, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Error marshalling JSON: %v", err)
				continue
			}

			err = kafkaWriter.WriteMessages(context.Background(), kafka.Message{
				Value: jsonMsg,
			})
			if err != nil {
				log.Printf("Could not write message to kafka: %v", err)
			} else {
				log.Printf("Wrote message to Kafka: %s", string(jsonMsg))
			}

			time.Sleep(2 * time.Second)
		}
	}()
}

func main() {
	kafkaBroker := "kafka:29092"
	topic := "metrics-raw"
	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	recordMetrics(writer)
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Beginning to serve on port :9100")
	log.Fatal(http.ListenAndServe(":9100", nil))
}
