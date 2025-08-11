package main

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

func recordMetrics() {
	go func() {
		for {
			cpuPercentages, err := cpu.Percent(time.Second, false)
			if err != nil {
				log.Printf("Error getting CPU usage: %v", err)
			} else if len(cpuPercentages) > 0 {
				cpuUsage.Set(cpuPercentages[0])
			}

			vMem, err := mem.VirtualMemory()
			if err != nil {
				log.Printf("Error getting memory usage: %v", err)
			} else {
				memUsage.Set(vMem.UsedPercent)
			}

			time.Sleep(2 * time.Second)
		}
	}()
}

func main() {
	recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Beginning to serve on port :9100")

	log.Fatal(http.ListenAndServe(":9100", nil))
}
