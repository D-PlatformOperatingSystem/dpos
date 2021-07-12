package metrics

import (
	"time"

	dplatformoslog "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/metrics/influxdb"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	go_metrics "github.com/rcrowley/go-metrics"
)

type influxDBPara struct {
	//
	Duration  int64  `json:"duration,omitempty"`
	URL       string `json:"url,omitempty"`
	Database  string `json:"database,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

var (
	log = dplatformoslog.New("module", "dplatformos metrics")
)

//StartMetrics             m
func StartMetrics(cfg *types.DplatformOSConfig) {
	metrics := cfg.GetModuleConfig().Metrics
	if !metrics.EnableMetrics {
		log.Info("Metrics data is not enabled to emit")
		return
	}

	switch metrics.DataEmitMode {
	case "influxdb":
		sub := cfg.GetSubConfig().Metrics
		subcfg, ok := sub[metrics.DataEmitMode]
		if !ok {
			log.Error("nil parameter for influxdb")
		}
		var influxdbcfg influxDBPara
		types.MustDecode(subcfg, &influxdbcfg)
		log.Info("StartMetrics with influxdb", "influxdbcfg.Duration", influxdbcfg.Duration,
			"influxdbcfg.URL", influxdbcfg.URL,
			"influxdbcfg.DatabaseName,", influxdbcfg.Database,
			"influxdbcfg.Username", influxdbcfg.Username,
			"influxdbcfg.Password", influxdbcfg.Password,
			"influxdbcfg.Namespace", influxdbcfg.Namespace)
		go influxdb.InfluxDB(go_metrics.DefaultRegistry,
			time.Duration(influxdbcfg.Duration),
			influxdbcfg.URL,
			influxdbcfg.Database,
			influxdbcfg.Username,
			influxdbcfg.Password,
			"")
	default:
		log.Error("startMetrics", "The dataEmitMode set is not supported now ", metrics.DataEmitMode)
		return
	}
}
