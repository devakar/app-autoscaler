package integration

import (
	"autoscaler/cf"
	egConfig "autoscaler/eventgenerator/config"
	mcConfig "autoscaler/metricscollector/config"
	"autoscaler/models"
	seConfig "autoscaler/scalingengine/config"
	"encoding/json"
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit/ginkgomon"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

const (
	APIServer        = "apiServer"
	ServiceBroker    = "serviceBroker"
	Scheduler        = "scheduler"
	MetricsCollector = "metricsCollector"
	EventGenerator   = "eventGenerator"
	ScalingEngine    = "scalingEngine"
)

var testCertDir string = "../../test-certs"

type Executables map[string]string
type Ports map[string]int

type Components struct {
	Executables Executables
	Ports       Ports
}

type DBConfig struct {
	URI            string `json:"uri"`
	MinConnections int    `json:"minConnections"`
	MaxConnections int    `json:"maxConnections"`
	IdleTimeout    int    `json:"idleTimeout"`
}

type ServiceBrokerConfig struct {
	Port int `json:"port"`

	Username string `json:"username"`
	Password string `json:"password"`

	DB DBConfig `json:"db"`

	APIServerUri       string          `json:"apiServerUri"`
	HttpRequestTimeout int             `json:"httpRequestTimeout"`
	TLS                models.TLSCerts `json:"tls"`
}

type APIServerConfig struct {
	Port int `json:"port"`

	DB DBConfig `json:"db"`

	SchedulerUri string `json:"schedulerUri"`

	TLS models.TLSCerts `json:"tls"`
}

func (components *Components) ServiceBroker(confPath string, argv ...string) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name:              ServiceBroker,
		AnsiColorCode:     "32m",
		StartCheck:        "Service broker app is running",
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			"node", append([]string{components.Executables[ServiceBroker], "-c", confPath}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}

func (components *Components) ApiServer(confPath string, argv ...string) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name:              APIServer,
		AnsiColorCode:     "33m",
		StartCheck:        "Autoscaler API server started",
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			"node", append([]string{components.Executables[APIServer], "-c", confPath}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}
func (components *Components) Scheduler(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              Scheduler,
		AnsiColorCode:     "34m",
		StartCheck:        "Started SchedulerApplication in",
		StartCheckTimeout: 60 * time.Second,
		Command: exec.Command(
			"java", append([]string{"-jar", "-Dspring.config.location=" + confPath, "-Dserver.port=" + strconv.FormatInt(int64(components.Ports[Scheduler]), 10), components.Executables[Scheduler]}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}
func (components *Components) MetricsCollector(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              MetricsCollector,
		AnsiColorCode:     "35m",
		StartCheck:        `"metricscollector.started"`,
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			components.Executables[MetricsCollector],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}

func (components *Components) EventGenerator(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              EventGenerator,
		AnsiColorCode:     "36m",
		StartCheck:        `"eventgenerator.started"`,
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			components.Executables[EventGenerator],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}
func (components *Components) ScalingEngine(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              ScalingEngine,
		AnsiColorCode:     "37m",
		StartCheck:        `"scalingengine.started"`,
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			components.Executables[ScalingEngine],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}
func (components *Components) PrepareServiceBrokerConfig(port int, username string, password string, dbUri string, apiServerUri string, brokerApiHttpRequestTimeout time.Duration, tmpDir string) string {
	brokerConfig := ServiceBrokerConfig{
		Port:     port,
		Username: username,
		Password: password,
		DB: DBConfig{
			URI:            dbUri,
			MinConnections: 1,
			MaxConnections: 10,
			IdleTimeout:    1000,
		},
		APIServerUri:       apiServerUri,
		HttpRequestTimeout: int(brokerApiHttpRequestTimeout / time.Millisecond),
		TLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "servicebroker.key"),
			CertFile:   filepath.Join(testCertDir, "servicebroker.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}

	cfgFile, err := ioutil.TempFile(tmpDir, ServiceBroker)
	w := json.NewEncoder(cfgFile)
	err = w.Encode(brokerConfig)
	Expect(err).NotTo(HaveOccurred())
	cfgFile.Close()
	return cfgFile.Name()
}

func (components *Components) PrepareApiServerConfig(port int, dbUri string, schedulerUri string, tmpDir string) string {
	apiConfig := APIServerConfig{
		Port: port,

		DB: DBConfig{
			URI:            dbUri,
			MinConnections: 1,
			MaxConnections: 10,
			IdleTimeout:    1000,
		},

		SchedulerUri: schedulerUri,

		TLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "api.key"),
			CertFile:   filepath.Join(testCertDir, "api.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}

	cfgFile, err := ioutil.TempFile(tmpDir, APIServer)
	w := json.NewEncoder(cfgFile)
	err = w.Encode(apiConfig)
	Expect(err).NotTo(HaveOccurred())
	cfgFile.Close()
	return cfgFile.Name()
}

func (components *Components) PrepareSchedulerConfig(dbUri string, scalingEngineUri string, tmpDir string) string {
	dbUrl, _ := url.Parse(dbUri)
	scheme := dbUrl.Scheme
	host := dbUrl.Host
	path := dbUrl.Path
	userInfo := dbUrl.User
	userName := userInfo.Username()
	password, _ := userInfo.Password()
	if scheme == "postgres" {
		scheme = "postgresql"
	}
	jdbcDBUri := fmt.Sprintf("jdbc:%s://%s%s", scheme, host, path)
	settingStrTemplate := `
#datasource for application and quartz
spring.datasource.driverClassName=org.postgresql.Driver
spring.datasource.url=%s
spring.datasource.username=%s
spring.datasource.password=%s
#quartz job
scalingenginejob.reschedule.interval.millisecond=10000
scalingenginejob.reschedule.maxcount=6
scalingengine.notification.reschedule.maxcount=3
# scaling engine url
autoscaler.scalingengine.url=%s
#ssl
server.ssl.key-store=%s/scheduler.p12
caCert=%s/autoscaler-ca.crt
server.ssl.key-alias=scheduler
server.ssl.key-store-password=123456
#server.ssl.key-password=123456
#server.ssl.key-store-type=P12
  `
	settingJonsStr := fmt.Sprintf(settingStrTemplate, jdbcDBUri, userName, password, scalingEngineUri, testCertDir, testCertDir)
	cfgFile, err := os.Create(filepath.Join(tmpDir, "integration.properties"))
	Expect(err).NotTo(HaveOccurred())
	ioutil.WriteFile(cfgFile.Name(), []byte(settingJonsStr), 0777)
	cfgFile.Close()
	return cfgFile.Name()
}

func (components *Components) PrepareMetricsCollectorConfig(dbUri string, port int, ccNOAAUAAUrl string, cfGrantTypePassword string, pollInterval time.Duration, refreshInterval time.Duration, tmpDir string) string {
	cfg := mcConfig.Config{
		Cf: cf.CfConfig{
			Api:       ccNOAAUAAUrl,
			GrantType: cfGrantTypePassword,
			Username:  "admin",
			Password:  "admin",
		},
		Server: mcConfig.ServerConfig{
			Port: port,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metricscollector.key"),
				CertFile:   filepath.Join(testCertDir, "metricscollector.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Logging: mcConfig.LoggingConfig{
			Level: "debug",
		},
		Db: mcConfig.DbConfig{
			InstanceMetricsDbUrl: dbUri,
			PolicyDbUrl:          dbUri,
		},
		Collector: mcConfig.CollectorConfig{
			PollInterval:    pollInterval,
			RefreshInterval: refreshInterval,
		},
	}

	return writeYmlConfig(tmpDir, MetricsCollector, &cfg)
}

func (components *Components) PrepareEventGeneratorConfig(dbUri string, port int, metricsCollectorUrl string, scalingEngineUrl string, aggregatorExecuteInterval time.Duration, policyPollerInterval time.Duration, evaluationManagerInterval time.Duration, tmpDir string) string {
	conf := &egConfig.Config{
		Server: egConfig.ServerConfig{
			Port: port,
		},
		Logging: egConfig.LoggingConfig{
			Level: "debug",
		},
		Aggregator: egConfig.AggregatorConfig{
			AggregatorExecuteInterval: aggregatorExecuteInterval,
			PolicyPollerInterval:      policyPollerInterval,
			MetricPollerCount:         1,
			AppMonitorChannelSize:     1,
		},
		Evaluator: egConfig.EvaluatorConfig{
			EvaluationManagerInterval: evaluationManagerInterval,
			EvaluatorCount:            1,
			TriggerArrayChannelSize:   1,
		},
		DB: egConfig.DBConfig{
			PolicyDBUrl:    dbUri,
			AppMetricDBUrl: dbUri,
		},
		ScalingEngine: egConfig.ScalingEngineConfig{
			ScalingEngineUrl: scalingEngineUrl,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		MetricCollector: egConfig.MetricCollectorConfig{
			MetricCollectorUrl: metricsCollectorUrl,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
	}
	return writeYmlConfig(tmpDir, EventGenerator, &conf)
}

func (components *Components) PrepareScalingEngineConfig(dbUri string, port int, ccUAAUrl string, cfGrantTypePassword string, tmpDir string) string {
	conf := seConfig.Config{
		Cf: cf.CfConfig{
			Api:       ccUAAUrl,
			GrantType: cfGrantTypePassword,
			Username:  "admin",
			Password:  "admin",
		},
		Server: seConfig.ServerConfig{
			Port: port,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scalingengine.key"),
				CertFile:   filepath.Join(testCertDir, "scalingengine.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Logging: seConfig.LoggingConfig{
			Level: "debug",
		},
		Db: seConfig.DbConfig{
			PolicyDbUrl:        dbUri,
			ScalingEngineDbUrl: dbUri,
			SchedulerDbUrl:     dbUri,
		},
		Synchronizer: seConfig.SynchronizerConfig{
			ActiveScheduleSyncInterval: 10 * time.Minute,
		},
	}

	return writeYmlConfig(tmpDir, ScalingEngine, &conf)
}

func writeYmlConfig(dir string, componentName string, c interface{}) string {
	cfgFile, err := ioutil.TempFile(dir, componentName)
	Expect(err).NotTo(HaveOccurred())
	defer cfgFile.Close()
	configBytes, err := yaml.Marshal(c)
	ioutil.WriteFile(cfgFile.Name(), configBytes, 0777)
	return cfgFile.Name()

}
