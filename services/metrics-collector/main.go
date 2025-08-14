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
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

var (
	cpuUsage   = promauto.NewGauge(prometheus.GaugeOpts{Name: "node_cpu_usage_percent"})
	memUsage   = promauto.NewGauge(prometheus.GaugeOpts{Name: "node_memory_usage_percent"})
	load1      = promauto.NewGauge(prometheus.GaugeOpts{Name: "node_load1"})
	diskReads  = promauto.NewGauge(prometheus.GaugeOpts{Name: "node_disk_reads_per_second"})
	diskWrites = promauto.NewGauge(prometheus.GaugeOpts{Name: "node_disk_writes_per_second"})
	netSent    = promauto.NewGauge(prometheus.GaugeOpts{Name: "node_net_sent_bytes_per_second"})
	netRecv    = promauto.NewGauge(prometheus.GaugeOpts{Name: "node_net_recv_bytes_per_second"})
)

type MetricMessage struct {
	Hostname          string    `json:"hostname"`
	Timestamp         time.Time `json:"timestamp"`
	CPU_Usage         float64   `json:"cpu_usage_percent"`
	MEM_Usage         float64   `json:"mem_usage_percent"`
	Load_1            float64   `json:"load_1"`
	Disk_Reads_PS     float64   `json:"disk_reads_ps"`
	Disk_Writes_PS    float64   `json:"disk_writes_ps"`
	Net_Sent_Bytes_PS float64   `json:"net_sent_bytes_ps"`
	Net_Recv_Bytes_PS float64   `json:"net_recv_bytes_ps"`
}

func recordMetrics(kafkaWriter *kafka.Writer) {
	hostname, _ := os.Hostname()
	var lastNetCounters net.IOCountersStat
	var lastDiskCounters disk.IOCountersStat
	var lastTime time.Time

	go func() {
		for {
			currentTime := time.Now()
			cpuVal, _ := cpu.Percent(time.Second, false)
			memVal, _ := mem.VirtualMemory()
			loadVal, _ := load.Avg()
			netCounters, _ := net.IOCounters(false)
			diskCounters, _ := disk.IOCounters()

			if !lastTime.IsZero() {
				duration := currentTime.Sub(lastTime).Seconds()

				diskReadRate := float64(diskCounters["sda"].ReadCount-lastDiskCounters.ReadCount) / duration
				diskWriteRate := float64(diskCounters["sda"].WriteCount-lastDiskCounters.WriteCount) / duration
				netSentRate := float64(netCounters[0].BytesSent-lastNetCounters.BytesSent) / duration
				netRecvRate := float64(netCounters[0].BytesRecv-lastNetCounters.BytesRecv) / duration

				cpuUsage.Set(cpuVal[0])
				memUsage.Set(memVal.UsedPercent)
				load1.Set(loadVal.Load1)
				diskReads.Set(diskReadRate)
				diskWrites.Set(diskWriteRate)
				netSent.Set(netSentRate)
				netRecv.Set(netRecvRate)

				msg := MetricMessage{
					Hostname:          hostname,
					Timestamp:         currentTime,
					CPU_Usage:         cpuVal[0],
					MEM_Usage:         memVal.UsedPercent,
					Load_1:            loadVal.Load1,
					Disk_Reads_PS:     diskReadRate,
					Disk_Writes_PS:    diskWriteRate,
					Net_Sent_Bytes_PS: netSentRate,
					Net_Recv_Bytes_PS: netRecvRate,
				}
				jsonMsg, _ := json.Marshal(msg)
				kafkaWriter.WriteMessages(context.Background(), kafka.Message{Value: jsonMsg})
			}

			lastNetCounters = netCounters[0]
			lastDiskCounters = diskCounters["sda"]
			lastTime = currentTime
			time.Sleep(4 * time.Second)
		}
	}()
}
func main() {
	log.Println("Agent service starting... waiting 15 seconds for Kafka to be ready.")
	time.Sleep(15 * time.Second)

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
