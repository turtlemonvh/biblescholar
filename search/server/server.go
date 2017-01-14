package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Jeffail/gabs"
	log "github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	//"github.com/rcrowley/go-metrics/exp"
	"gopkg.in/tylerb/graceful.v1"
)

type ServerConfig struct {
	Port int
	// Include index hash here too
	BuildCommit string
	BuildBranch string
	Index       bleve.Index
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

// Return version status information
func versionHandler(s *ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, map[string]interface{}{
			"commit": s.BuildCommit,
			"branch": s.BuildBranch,
		})
	}
}

// https://developer.amazon.com/public/solutions/alexa/alexa-skills-kit/docs/alexa-skills-kit-interface-reference
const exampleAlexaRequest string = `
{
	"version": "1.0",
	"session": {
		"new": true,
		"sessionId": "string",
		"application": {
			"applicationId": "string"
		},
		"attributes": {
			"string": {}
		},
		"user": {
			"userId": "string",
			"accessToken": "string"
		}
	},
	"context": {
		"System": {
			"application": {
				"applicationId": "string"
			},
			"user": {
				"userId": "string",
				"accessToken": "string"
			},
			"device": {
				"supportedInterfaces": {
					"AudioPlayer": {}
				}
			}
		},
		"AudioPlayer": {
			"token": "string",
			"offsetInMilliseconds": 0,
			"playerActivity": "string"
		}
	},
	"request": {
		"type": "IntentRequest",
		"requestId": "string",
		"timestamp": "2016-12-38T00:00",
		"locale": "en-US",
		"intent": {
			"name": "SearchBible",
			"slots": {
				"QueryPhrase": {
					"name": "QueryPhrase",
					"value": "for God so loved the world"
				}
			}
		}
	}
}
`

// https://developer.amazon.com/public/solutions/alexa/alexa-skills-kit/docs/alexa-skills-kit-interface-reference#response-format
var respTemplate string = `
{
  "version": "string",
  "sessionAttributes": {
    "string": object
  },
  "response": {
    "outputSpeech": {
      "type": "string",
      "text": "string",
      "ssml": "string"
    },
    "card": {
      "type": "string",
      "title": "string",
      "content": "string",
      "text": "string",
      "image": {
        "smallImageUrl": "string",
        "largeImageUrl": "string"
      }
    },
    "reprompt": {
      "outputSpeech": {
        "type": "string",
        "text": "string",
        "ssml": "string"
      }
    },
    "directives": [
      {
        "type": "string",
        "playBehavior": "string",
        "audioItem": {
          "stream": {
            "token": "string",
            "url": "string",
            "offsetInMilliseconds": 0
          }
        }
      }
    ],
    "shouldEndSession": boolean
  }
}
`

// https://developer.amazon.com/public/solutions/alexa/alexa-skills-kit/docs/alexa-skills-kit-interface-reference#response-object
func (s *ServerConfig) getNewResponseTemplate() *gabs.Container {
	alexResp := gabs.New()
	alexResp.SetP(s.VersionString(), "version")
	alexResp.SetP(map[string]interface{}{}, "sessionAttributes")

	// Eventually if they want more results and keep the session open
	alexResp.SetP(true, "shouldEndSession")

	// Remove unused
	alexResp.DeleteP("response.outputSpeech")
	alexResp.DeleteP("response.reprompt")
	alexResp.DeleteP("response.directives")
	alexResp.DeleteP("response.card.image")
	alexResp.DeleteP("response.card.text")

	return alexResp
}

// Probably don't have to return the container object
func setResponseText(ro *gabs.Container, txt string, title string, reprompt bool) error {
	responsePath := "outputSpeech"
	if reprompt {
		responsePath = "reprompt"
	}

	ro.SetP(map[string]interface{}{
		"type": "PlainText",
		"text": txt,
	}, fmt.Sprintf("response.%s", responsePath))

	ro.SetP("Simple", "response.card.type")
	ro.SetP(title, "response.card.title")
	ro.SetP(txt, "response.card.content")

	return nil
}

// Respond to search requests
// FIXME: Add tests
// FIXME: Break down into smaller functions
func alexaSearchHandler(s *ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
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
