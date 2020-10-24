package nebula

import (
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/arl/statsviz"
	"go.uber.org/atomic"
)

// profileServer allows serving go runtime debugging
// and profiling information, and allows serving
// the statsviz runtime visualizer, as well as net/http/pprof handlers
type profileServer struct {
	sync.Mutex
	start   *sync.Once
	running *atomic.Bool
	stop    chan struct{}
}

func newProfileServer() *profileServer {
	return &profileServer{
		start:   &sync.Once{},
		running: atomic.NewBool(false),
		stop:    make(chan struct{}),
	}
}

func (p *profileServer) Start(addr string, disablePprof, disableStatsViz bool) {
	if disablePprof && disableStatsViz {
		l.Error("pprof and statsviz disabled")
		return
	}
	// prevent a panic
	if p.start == nil {
		l.Error("statsvizServer object not properly initialized")
		return
	}
	p.Lock()
	defer p.Unlock()
	p.start.Do(func() {
		p.running.Store(true)
		mux := http.NewServeMux()
		if !disablePprof {
			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		}
		if !disableStatsViz {
			statsviz.Register(mux)
		}
		srv := &http.Server{
			Addr:    addr,
			Handler: mux,
		}
		go srv.ListenAndServe()
		go func() {
			<-p.stop
			srv.Close()
			p.running.Store(false)
		}()
	})
}

// used to stop the statsviz server but prepares the struct
// for being able to startup another server
func (p *profileServer) Reset() {
	if !p.running.Load() {
		return
	}
	p.Stop()
	for {
		if p.running.Load() {
			time.Sleep(time.Microsecond * 250)
			continue
		}
		break
	}
	p.Lock()
	defer p.Unlock()
	p.start = &sync.Once{}
}

// note: tthis will block, maybe we should do a non-blocking send?
func (p *profileServer) Stop() {
	p.stop <- struct{}{}
}
