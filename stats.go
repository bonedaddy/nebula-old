package nebula

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/arl/statsviz"
	graphite "github.com/cyberdelia/go-metrics-graphite"
	mp "github.com/nbrownus/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/atomic"
)

// helper object to start up a statsviz server
// primarily intended for use with the sshd management port
// however it could potentially be started at node startup as well
type statsvizServer struct {
	sync.Mutex
	start   *sync.Once
	running *atomic.Bool
	stop    chan struct{}
}

func newStatsViz() *statsvizServer {
	return &statsvizServer{
		start:   &sync.Once{},
		running: atomic.NewBool(false),
		stop:    make(chan struct{}),
	}
}

func (s *statsvizServer) Start(addr string) {
	// prevent a panic
	if s.start == nil {
		l.Error("statsvizServer object not properly initialized")
		return
	}
	s.Lock()
	defer s.Unlock()
	s.start.Do(func() {
		s.running.Store(true)
		mux := http.NewServeMux()
		statsviz.Register(mux)
		srv := &http.Server{
			Addr:    addr,
			Handler: mux,
		}
		go srv.ListenAndServe()
		go func() {
			<-s.stop
			srv.Close()
			s.running.Store(false)
		}()
	})
}

// used to stop the statsviz server but prepares the struct
// for being able to startup another server
func (s *statsvizServer) Reset() {
	if !s.running.Load() {
		return
	}
	s.Stop()
	for {
		if s.running.Load() {
			time.Sleep(time.Microsecond * 250)
			continue
		}
		break
	}
	s.Lock()
	defer s.Unlock()
	s.start = &sync.Once{}
}

// note: tthis will block, maybe we should do a non-blocking send?
func (s *statsvizServer) Stop() {
	s.stop <- struct{}{}
}

func startStats(c *Config, configTest bool) error {
	mType := c.GetString("stats.type", "")
	if mType == "" || mType == "none" {
		return nil
	}

	interval := c.GetDuration("stats.interval", 0)
	if interval == 0 {
		return fmt.Errorf("stats.interval was an invalid duration: %s", c.GetString("stats.interval", ""))
	}

	switch mType {
	case "graphite":
		startGraphiteStats(interval, c, configTest)
	case "prometheus":
		startPrometheusStats(interval, c, configTest)
	default:
		return fmt.Errorf("stats.type was not understood: %s", mType)
	}

	metrics.RegisterDebugGCStats(metrics.DefaultRegistry)
	metrics.RegisterRuntimeMemStats(metrics.DefaultRegistry)

	go metrics.CaptureDebugGCStats(metrics.DefaultRegistry, interval)
	go metrics.CaptureRuntimeMemStats(metrics.DefaultRegistry, interval)

	return nil
}

func startGraphiteStats(i time.Duration, c *Config, configTest bool) error {
	proto := c.GetString("stats.protocol", "tcp")
	host := c.GetString("stats.host", "")
	if host == "" {
		return errors.New("stats.host can not be empty")
	}

	prefix := c.GetString("stats.prefix", "nebula")
	addr, err := net.ResolveTCPAddr(proto, host)
	if err != nil {
		return fmt.Errorf("error while setting up graphite sink: %s", err)
	}

	l.Sugar().Infof("Starting graphite. Interval: %s, prefix: %s, addr: %s", i, prefix, addr)
	if !configTest {
		go graphite.Graphite(metrics.DefaultRegistry, i, prefix, addr)
	}
	return nil
}

func startPrometheusStats(i time.Duration, c *Config, configTest bool) error {
	namespace := c.GetString("stats.namespace", "")
	subsystem := c.GetString("stats.subsystem", "")

	listen := c.GetString("stats.listen", "")
	if listen == "" {
		return fmt.Errorf("stats.listen should not be empty")
	}

	path := c.GetString("stats.path", "")
	if path == "" {
		return fmt.Errorf("stats.path should not be empty")
	}

	pr := prometheus.NewRegistry()
	pClient := mp.NewPrometheusProvider(metrics.DefaultRegistry, namespace, subsystem, pr, i)
	go pClient.UpdatePrometheusMetrics()

	if !configTest {
		go func() {
			l.Sugar().Infof("Prometheus stats listening on %s at %s", listen, path)
			http.Handle(path, promhttp.HandlerFor(pr, promhttp.HandlerOpts{}))
			log.Fatal(http.ListenAndServe(listen, nil))
		}()
	}

	return nil
}
