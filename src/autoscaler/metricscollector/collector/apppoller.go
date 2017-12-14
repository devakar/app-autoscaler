package collector

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/metricscollector/noaa"
	"autoscaler/models"
	"fmt"
	"strconv"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry/sonde-go/events"

	"time"
)

type appPoller struct {
	appId           string
	appMetricDB     db.AppMetricDB
	collectInterval time.Duration
	logger          lager.Logger
	cfc             cf.CfClient
	noaaConsumer    noaa.NoaaConsumer
	pclock          clock.Clock
	doneChan        chan bool
	dataChan        chan *models.AppInstanceMetric
}

func NewAppPoller(logger lager.Logger, appId string, appMetricDB db.AppMetricDB, collectInterval time.Duration, cfc cf.CfClient, noaaConsumer noaa.NoaaConsumer, pclock clock.Clock, dataChan chan *models.AppInstanceMetric) AppCollector {
	return &appPoller{
		appId:           appId,
		appMetricDB:     appMetricDB,
		collectInterval: collectInterval,
		logger:          logger,
		cfc:             cfc,
		noaaConsumer:    noaaConsumer,
		pclock:          pclock,
		doneChan:        make(chan bool),
		dataChan:        dataChan,
	}

}

func (ap *appPoller) Start() {
	go ap.startPollMetrics()

	ap.logger.Info("app-poller-started", lager.Data{"appid": ap.appId, "collect-interval": ap.collectInterval})
}

func (ap *appPoller) Stop() {
	ap.doneChan <- true
	ap.logger.Info("app-poller-stopped", lager.Data{"appid": ap.appId})
}

func (ap *appPoller) startPollMetrics() {
	for {
		ap.pollMetric()
		timer := ap.pclock.NewTimer(ap.collectInterval)
		select {
		case <-ap.doneChan:
			timer.Stop()
			return
		case <-timer.C():
		}
	}
}

func (ap *appPoller) pollMetric() {
	logger := ap.logger.WithData(lager.Data{"appId": ap.appId})
	logger.Debug("poll-metric")

	var containerEnvelopes []*events.Envelope
	var err error

	for attempt := 0; attempt < 3; attempt++ {
		logger.Debug("poll-metric-from-noaa-retry", lager.Data{"attempt": attempt + 1})
		containerEnvelopes, err = ap.noaaConsumer.ContainerEnvelopes(ap.appId, cf.TokenTypeBearer+" "+ap.cfc.GetTokens().AccessToken)
		if err == nil {
			break
		}
	}
	if err != nil {
		logger.Error("poll-metric-from-noaa", err)
		return
	}

	logger.Debug("poll-metric-get-containerenvelopes", lager.Data{"envelops": containerEnvelopes})

	metrics := noaa.GetInstanceMemoryMetricsFromContainerEnvelopes(ap.pclock.Now().UnixNano(), ap.appId, containerEnvelopes)
	logger.Debug("poll-metric-get-memory-metrics", lager.Data{"metrics": metrics})

	var count int64 = 0
	var sum int64 = 0
	var unit string
	var metricType string
	var timestamp int64 = time.Now().UnixNano()
	for _, metric := range metrics {
		metric_for_aggregation := metric
		ap.dataChan <- metric

		unit = metric.Unit
		metricType = metric.Name
		metricValue, err := strconv.ParseInt(metric.Value, 10, 64)
		if err != nil {
			m.logger.Error("failed-to-aggregate", err, lager.Data{"appid": ap.appId, "metrictype": metricType, "value": metric.Value})
		} else {
			count++
			sum += metricValue
		}
	}
	var appMetric *models.AppMetric
	if count == 0 {
		appMetric = &models.AppMetric{
			AppId:      ap.appId,
			MetricType: metricType,
			Value:      "",
			Unit:       "",
			Timestamp:  timestamp,
		}
	}

	avgValue := int64(float64(sum)/float64(count) + 0.5)
	appMetric = &models.AppMetric{
		AppId:      ap.appId,
		MetricType: metricType,
		Value:      fmt.Sprintf("%d", avgValue),
		Unit:       unit,
		Timestamp:  timestamp,
	}

	err = ap.appMetricDB.SaveAppMetric(avgMetric)
	if err != nil {
		m.logger.Error("Failed to save appmetric", err, lager.Data{"appmetric": avgMetric})
	}
}
