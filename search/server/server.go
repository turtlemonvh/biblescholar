package server

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/Jeffail/gabs"
	log "github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	//"github.com/rcrowley/go-metrics/exp"
	"gopkg.in/tylerb/graceful.v1"
)

var (
	nAlexRequests metrics.Counter
	nRestRequests metrics.Counter
)

func init() {
	nAlexRequests = metrics.NewCounter()
	metrics.Register("nAlexRequests", nAlexRequests)

	nRestRequests = metrics.NewCounter()
	metrics.Register("nRestRequests", nRestRequests)

	// Respond with a function call every time they are called
	metrics.NewRegisteredFunctionalGauge("ngoroutines", metrics.DefaultRegistry, func() int64 { return int64(runtime.NumGoroutine()) })
	metrics.NewRegisteredFunctionalGauge("ncgocalls", metrics.DefaultRegistry, func() int64 { return int64(runtime.NumCgoCall()) })
	metrics.NewRegisteredFunctionalGauge("ncpus", metrics.DefaultRegistry, func() int64 { return int64(runtime.NumCPU()) })
}

type ServerConfig struct {
	Port int
	// Include index hash here too
	BuildCommit         string
	BuildBranch         string
	Index               bleve.Index
	ShouldValidateAlexa bool
}

func (s *ServerConfig) VersionString() string {
	return fmt.Sprintf("%s (%s)", s.BuildBranch, s.BuildCommit)
}

func (s *ServerConfig) StartServer() {
	log.WithFields(log.Fields{
		"port":   s.Port,
		"commit": s.BuildCommit,
		"branch": s.BuildBranch,
	}).Info("Starting server")

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(ginrus.Ginrus(log.StandardLogger(), time.RFC3339, true))
	r.Use(gin.Recovery())
	r.GET("/", versionHandler(s))
	r.GET("/opt/version", versionHandler(s))
	r.GET("/opt/metrics", gin.WrapH(exp.ExpHandler(metrics.DefaultRegistry)))
	r.GET("/search", searchHandler(s))
	r.POST("/alexa/search", alexaSearchHandler(s))

	log.WithFields(log.Fields{
		"port": s.Port,
	}).Info("Starting server")

	srv := &graceful.Server{
		Timeout: 10 * time.Second,
		Server: &http.Server{
			Addr:    fmt.Sprintf(":%d", s.Port),
			Handler: r,
		},
		BeforeShutdown: func() bool {
			log.Warn("Starting shutdown.")
			return true
		},
	}
	srv.ListenAndServe()

	log.Warn("Everything safely closed. Exiting main process.")
}

// Top level handlers

// Return version status information
func versionHandler(s *ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := make(map[string]interface{})
		status["commit"] = s.BuildCommit
		status["branch"] = s.BuildBranch
		c.JSON(200, status)
	}
}

// Respond to search requests
// FIXME: Add tests
// FIXME: Break down into smaller functions
func alexaSearchHandler(s *ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		nAlexRequests.Inc(1)

		if s.ShouldValidateAlexa {
			if err := s.verifyRequestIsAlexa(c); err != nil {
				// Actual response is set inside this function
				return
			}
		}

		resp := s.getNewResponseTemplate()

		req, err := gabs.ParseJSONBuffer(c.Request.Body)
		if err != nil || req == nil {
			log.WithFields(log.Fields{
				"err": err,
				"req": req,
			}).Error("Error while parsing request body")
			c.JSON(http.StatusBadRequest, resp.Data())
			return
		}

		if s.ShouldValidateAlexa {
			if err := s.validateAppId(c, req); err != nil {
				// Actual response is set inside this function
				return
			}
		}

		t, err := getRequestType(req)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("Error getting request type")
			c.JSON(http.StatusBadRequest, resp.Data())
			return
		}

		switch t {
		case intentRequest:
			s.handleIntentRequest(c, req, resp)
		case launchRequest:
			s.handleLaunchRequest(c, req, resp)
		case sessionEndedRequest:
			s.handleSessionEndedRequest(c, req, resp)
		default:
			log.WithFields(log.Fields{
				"type": t,
			}).Error("Unknown request type")
			c.JSON(http.StatusBadRequest, resp.Data())
		}
	}
}
