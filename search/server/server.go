package server

import (
	"fmt"
	"html/template"
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
	nHtmlRequests metrics.Counter
)

func init() {
	nAlexRequests = metrics.NewCounter()
	metrics.Register("nAlexRequests", nAlexRequests)

	nRestRequests = metrics.NewCounter()
	metrics.Register("nRestRequests", nRestRequests)

	nHtmlRequests = metrics.NewCounter()
	metrics.Register("nHtmlRequests", nHtmlRequests)

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
	template            *template.Template
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

	var err error
	s.template, err = template.New("ServerTemplate").Parse(templateSource)
	if err != nil {
		panic(err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(ginrus.Ginrus(log.StandardLogger(), time.RFC3339, true))
	r.Use(gin.Recovery())
	r.GET("/", htmlHandler(s))
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

// Home page
const templateSource string = `
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8" />
<title>{{ $.Title }}</title>
<style type="text/css">
#results {
    list-style: none;
    padding-left: 2px;
}
#results li {
    padding-bottom: 1em;
}
#results li div[name=chapter-verse] {
	display: inline
}
</style>
</head>
<body>
	<h2>{{ $.Headline }}</h2>
	<div id="help">Query language reference: <a href="http://godoc.org/github.com/blevesearch/bleve#NewQueryStringQuery">bleve</a></div>
	<form action="/" method="GET">
	<input type="text" name="q">
    <br>
    <input type="submit" class="button" value="query string">
	</form>
{{ if $.ReturnResults }}
	<hr>
	<ul id="results">
	{{range $message := $.Hits }}
	<li>
		<span name="book">{{ $message.Fields.Book }}</span>
		<div name="chapter-verse">
			<span name="chapter">{{ $message.Fields.Chapter }}</span> : <span name="verse">{{ $message.Fields.Verse }}</span>
		</div>
		(<span name="version">{{ $message.Fields.Version }}</span>)
		<span name="text">{{ $message.Fields.Text }}</span>
	</li>
	{{ end }}
	</ul>
{{ end }}
</body>
</html>
`
