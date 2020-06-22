package metrics

import (
	"datamon/dbcheck"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"strconv"
)

type DatamonCollector struct {
	datamonDesc *prometheus.Desc
}

func (c *DatamonCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.datamonDesc
}

func (c *DatamonCollector) Collect(ch chan<- prometheus.Metric) {
	metricsQueue := dbcheck.GetQueue()
	for len(metricsQueue) > 0 {
		metric := <-metricsQueue
		value, err := strconv.ParseFloat(metric[2], 64)
		if err != nil {
			log.Printf("Failed to parse metrics value with error: %v", err)
		}
		checkName := metric[0]
		checkType := metric[1]
		ch <- prometheus.MustNewConstMetric(
			c.datamonDesc,
			prometheus.UntypedValue,
			value,
			checkName, checkType,
		)
	}
}

func NewDatamonCollector() *DatamonCollector {
	return &DatamonCollector{
		datamonDesc: prometheus.NewDesc("datamon_check_metric", "DB check metrics", []string{"name", "type"}, nil),
	}
}
