package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/locket"

	"github.com/hashicorp/consul/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"autoscaler/cf"
	"autoscaler/metricscollector"
	"autoscaler/metricscollector/config"

	"code.cloudfoundry.org/consuladapter"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("MetricsCollector", func() {
	var (
		runner       *MetricsCollectorRunner
		runner1      *MetricsCollectorRunner
		runner2      *MetricsCollectorRunner
		consulClient consuladapter.Client
	)
	consulIsPresent := false

	BeforeEach(func() {
		consulRunner.Reset()
		consulClient = consulRunner.NewClient()
		runner = NewMetricsCollectorRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	if consulIsPresent {
		Context("when the metricscollector acquires the lock", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.Start()

				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
			})

			It("registers itself with consul", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.registration-runner.succeeded-registering-service"))

				services, err := consulClient.Agent().Services()
				Expect(err).ToNot(HaveOccurred())

				Expect(services).To(HaveKeyWithValue("metricscollector",
					&api.AgentService{
						Service: "metricscollector",
						ID:      "metricscollector",
						Port:    cfg.Server.Port,
					}))
			})

			It("registers a TTL healthcheck", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.registration-runner.succeeded-registering-service"))

				checks, err := consulClient.Agent().Checks()
				Expect(err).ToNot(HaveOccurred())

				Expect(checks).To(HaveKeyWithValue("service:metricscollector",
					&api.AgentCheck{
						Node:        "0",
						CheckID:     "service:metricscollector",
						Name:        "Service 'metricscollector' check",
						Status:      "passing",
						ServiceID:   "metricscollector",
						ServiceName: "metricscollector",
					}))
			})

			It("should start", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.collector.collector-started"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})
		})

		Context("when the metricscollector loses the lock", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.Start()

				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))

				consulRunner.Reset()
			})

			It("exits with failure", func() {
				Eventually(runner.Session.Buffer, 4*time.Second).Should(gbytes.Say("exited-with-failure"))
				Eventually(runner.Session).Should(Exit(1))
			})
		})

		Context("when the metricscollector initially does not have the lock", func() {
			var competingMetricsCollectorProcess ifrit.Process

			BeforeEach(func() {
				logger := lagertest.NewTestLogger("competing-process")
				buffer := logger.Buffer()

				competingMetricsCollectorLock := locket.NewLock(logger, consulClient, metricscollector.MetricsCollectorLockSchemaPath(), []byte{}, clock.NewClock(), cfg.Lock.LockRetryInterval, cfg.Lock.LockTTL)
				competingMetricsCollectorProcess = ifrit.Invoke(competingMetricsCollectorLock)
				Eventually(buffer, 2*time.Second).Should(gbytes.Say("competing-process.lock.acquire-lock-succeeded"))

				runner.startCheck = ""
				runner.Start()
			})

			It("should not start", func() {
				Consistently(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("metricscollector.collector.collector-started"))
				Consistently(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("metricscollector.registration-runner"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.lock.acquiring-lock"))
				Consistently(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("metricscollector.started"))
			})

			Describe("when the lock becomes available", func() {
				BeforeEach(func() {
					ginkgomon.Kill(competingMetricsCollectorProcess)
				})

				It("acquires the lock and starts", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
					Consistently(runner.Session).ShouldNot(Exit())
				})

			})
		})

		Context("with a missing config file", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.configPath = "bogus"
				runner.Start()
			})

			It("fails with an error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to open config file"))
			})
		})

		Context("with an invalid config file", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				badfile, err := ioutil.TempFile("", "bad-mc-config")
				Expect(err).NotTo(HaveOccurred())
				runner.configPath = badfile.Name()
				ioutil.WriteFile(runner.configPath, []byte("bogus"), os.ModePerm)
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("fails with an error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to read config file"))
			})
		})

		Context("with missing configuration", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				missingConfig := cfg
				missingConfig.Cf = cf.CfConfig{
					Api: ccNOAAUAA.URL(),
				}

				missingConfig.Server.Port = 7000 + GinkgoParallelNode()
				missingConfig.Logging.Level = "debug"
				runner.configPath = writeConfig(&missingConfig).Name()
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should fail validation", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to validate configuration"))
			})
		})

		Context("when no consul is configured", func() {
			BeforeEach(func() {
				noConsulConf := cfg
				noConsulConf.Lock.ConsulClusterConfig = ""
				runner.configPath = writeConfig(&noConsulConf).Name()
				runner.startCheck = ""
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should not get metricscollector service", func() {
				Eventually(func() map[string]*api.AgentService {
					services, err := consulClient.Agent().Services()
					Expect(err).ToNot(HaveOccurred())
					return services
				}).ShouldNot(HaveKey("metricscollector"))
			})

			It("should start", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

		})

		Context("when an interrupt is sent", func() {
			BeforeEach(func() {
				runner.Start()
			})

			It("should stop", func() {
				runner.Session.Interrupt()
				Eventually(runner.Session, 5).Should(Exit(0))
			})
		})

		Describe("MetricsCollector REST API", func() {
			Context("when a request for memory metrics comes", func() {
				Context("when token is not expired", func() {
					BeforeEach(func() {
						eLock.Lock()
						isTokenExpired = false
						eLock.Unlock()
						runner.Start()
					})

					It("returns with a 200", func() {
						rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/metrics/memoryused", mcPort))
						Expect(err).NotTo(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusOK))
						rsp.Body.Close()
					})
				})

				Context("when token is expired", func() {
					BeforeEach(func() {
						eLock.Lock()
						isTokenExpired = true
						eLock.Unlock()
						runner.Start()
					})
					It("refreshes the token and returns with a 200", func() {
						rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/metrics/memoryused", mcPort))
						Expect(err).NotTo(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusOK))
						rsp.Body.Close()
					})
				})
			})

			Context("when a request for metrics history comes", func() {
				Context("when using polling for metrics collection", func() {
					BeforeEach(func() {
						runner.Start()
					})

					It("returns with a 200", func() {
						rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/metric_histories/a-metric-type", mcPort))
						Expect(err).NotTo(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusOK))
						rsp.Body.Close()
					})
				})
				Context("when using streaming for metrics collection", func() {
					BeforeEach(func() {
						streamingCfg := cfg
						streamingCfg.Collector.CollectMethod = config.CollectMethodStreaming
						runner.configPath = writeConfig(&streamingCfg).Name()
						runner.Start()
					})

					AfterEach(func() {
						os.Remove(runner.configPath)
					})

					It("returns with a 200", func() {
						rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/metric_histories/a-metric-type", mcPort))
						Expect(err).NotTo(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusOK))
						rsp.Body.Close()
					})
				})
			})

		})
	} else {

		Describe("DB Lock", func() {
			BeforeEach(func() {
				// noConsulConf := cfg
				// noConsulConf.Lock.ConsulClusterConfig = ""
				// runner.configPath = writeConfig(&noConsulConf).Name()
				// runner.startCheck = ""
				//runner.Start()
			})
			Context("when metricscollector aquires lock for first time", func() {
				BeforeEach(func() {
					noConsulConf := cfg
					noConsulConf.Lock.ConsulClusterConfig = ""
					runner.configPath = writeConfig(&noConsulConf).Name()
					runner.startCheck = ""
					runner.Start()

				})
				AfterEach(func() {
					runner.Interrupt()
				})
				It("successfully aquire lock and start", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("lock acquired in fisrt attempt"))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
				})
			})
			Context("when metricscollector have the lock", func() {
				BeforeEach(func() {
					noConsulConf := cfg
					noConsulConf.Lock.ConsulClusterConfig = ""
					runner.configPath = writeConfig(&noConsulConf).Name()
					runner.startCheck = ""
					runner.Start()
				})
				AfterEach(func() {
					runner.Interrupt()
				})
				It("successfully retry lock", func() {
					Eventually(runner.Session.Buffer, 35*time.Second).Should(gbytes.Say("retrying-acquiring-lock"))

				})
			})
			Context("when interrupt occurs", func() {
				BeforeEach(func() {
					noConsulConf := cfg
					noConsulConf.Lock.ConsulClusterConfig = ""
					runner.configPath = writeConfig(&noConsulConf).Name()
					runner.startCheck = ""
					runner.Start()
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
				})
				It("successfully release lock and stop", func() {
					runner.Interrupt()
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("received interrupt signal"))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("successfully released lock"))
				})
			})
			Context("When one instance of metricscollector has lock the other will wait", func() {
				BeforeEach(func() {
					runner1 = NewMetricsCollectorRunner()
					noConsulConf := cfg
					noConsulConf.Lock.ConsulClusterConfig = ""
					noConsulConf.Server.Port = 9000
					runner1.configPath = writeConfig(&noConsulConf).Name()
					runner1.startCheck = ""
					runner1.Start()
					Eventually(runner1.Session.Buffer, 5*time.Second).Should(gbytes.Say("metricscollector.started"))

					runner2 = NewMetricsCollectorRunner()
					noConsulConfWithPort := cfg
					noConsulConfWithPort.Lock.ConsulClusterConfig = ""
					noConsulConfWithPort.Server.Port = 8000
					noConsulConfWithPort.DBLock.Owner = "second-owner"
					runner2.configPath = writeConfig(&noConsulConfWithPort).Name()
					runner2.startCheck = ""
					runner2.Start()

				})
				AfterEach(func() {
					runner2.Interrupt()
					runner2.KillWithFire()
					runner1.Interrupt()
					runner1.KillWithFire()
				})
				It("should not get lock in first attempt", func() {
					Consistently(runner2.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("lock acquired in fisrt attempt"))
				})
			})
			Context("when one instance of metricscollector stopped", func() {
				BeforeEach(func() {
					runner1 = NewMetricsCollectorRunner()
					noConsulConf := cfg
					noConsulConf.Lock.ConsulClusterConfig = ""
					noConsulConf.Server.Port = 9000
					runner1.configPath = writeConfig(&noConsulConf).Name()
					runner1.startCheck = ""
					runner1.Start()
					Eventually(runner1.Session.Buffer, 20*time.Second).Should(gbytes.Say("metricscollector.started"))

					runner2 = NewMetricsCollectorRunner()
					noConsulConfWithPort := cfg
					noConsulConfWithPort.Lock.ConsulClusterConfig = ""
					noConsulConfWithPort.Server.Port = 8000
					noConsulConfWithPort.DBLock.Owner = "second-owner"
					runner2.configPath = writeConfig(&noConsulConfWithPort).Name()
					runner2.startCheck = ""
					runner2.Start()
					Eventually(runner2.Session.Buffer, 30*time.Second).Should(gbytes.Say("waiting for lock"))
				})
				AfterEach(func() {
					runner2.Interrupt()
					runner2.KillWithFire()
					runner1.KillWithFire()
				})
				It("other instance of metricscollector aquires lock", func() {
					runner1.Interrupt()
					Eventually(runner1.Session.Buffer, 5*time.Second).Should(gbytes.Say("received interrupt signal"))
					Eventually(runner1.Session.Buffer, 5*time.Second).Should(gbytes.Say("successfully released lock"))
					Eventually(runner2.Session.Buffer, 30*time.Second).Should(gbytes.Say("successfully acquired lock"))
					Eventually(runner2.Session.Buffer, 30*time.Second).Should(gbytes.Say("metricscollector.started"))
				})
			})
		})
	}
})
