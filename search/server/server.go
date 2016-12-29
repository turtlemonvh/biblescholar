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

const (
	requestIntentNamePath string = "request.intent.name"
	requestQueryPath      string = "request.intent.slots.QueryPhrase.value"
)

func (s *ServerConfig) getNewResponseTemplate() *gabs.Container {
	alexResp := gabs.New()
	alexResp.SetP(s.VersionString(), "version")
	alexResp.SetP(map[string]interface{}{}, "sessionAttributes")

	// Eventually if they want more results and keep the session open
	alexResp.SetP(true, "shouldEndSession")

	alexResp.SetP(map[string]interface{}{
		"type": "PlainText",
	}, "response.outputSpeech")

	alexResp.SetP(map[string]interface{}{
		"type": "Simple",
	}, "response.card")

	return alexResp
}

// Probably don't have to return the container object
func setResponseText(ro *gabs.Container, txt string, title string) error {
	_, err := ro.SetP(txt, "response.outputSpeech.text")
	if err != nil {
		return err
	}
	_, err = ro.SetP(txt, "response.card.content")
	if err != nil {
		return err
	}
	_, err = ro.SetP(title, "response.card.title")
	if err != nil {
		return err
	}
	return nil
}

// Respond to search requests
// FIXME: Add tests
func alexaSearchHandler(s *ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := s.getNewResponseTemplate()
		if err := setResponseText(resp, "Your request is malformed. Oops.", "Processing Error"); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("Error while creating response body")
			c.JSON(500, resp.Data())
			return
		}

		jsonParsed, err := gabs.ParseJSONBuffer(c.Request.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("Error while parsing request body")
			c.JSON(500, resp.Data())
			return
		}
		if jsonParsed == nil {
			log.WithFields(log.Fields{
				"err": "Gabs object is nil",
			}).Error("Error while parsing request body")
			c.JSON(500, resp.Data())
			return
		}

		// Check request has data where we expect
		if !jsonParsed.ExistsP(requestIntentNamePath) {
			log.WithFields(log.Fields{
				"path": requestIntentNamePath,
			}).Error("Required path in request object does not exist")
			c.JSON(500, resp.Data())
			return
		}
		if !jsonParsed.ExistsP(requestQueryPath) {
			log.WithFields(log.Fields{
				"path": requestQueryPath,
			}).Error("Required path in request object does not exist")
			c.JSON(500, resp.Data())
			return
		}

		intent, ok := jsonParsed.Path(requestIntentNamePath).Data().(string)
		if !ok || intent != "SearchBible" {
			// We only handle this one intent
			log.WithFields(log.Fields{
				"intentReceived": intent,
				"intentExpected": "SearchBible",
			}).Error("Invalid task intent")
			c.JSON(500, resp.Data())
			return
		}

		queryText, ok := jsonParsed.Path(requestQueryPath).Data().(string)
		if !ok {
			log.WithFields(log.Fields{
				"queryValue": requestQueryPath,
			}).Error("Invalid query value. Expected string.")
			c.JSON(500, resp.Data())
			return
		}

		// query, limit, skip, explain
		query := bleve.NewQueryStringQuery(queryText)
		searchRequest := bleve.NewSearchRequestOptions(query, 10, 0, false)
		searchRequest.Fields = []string{
			"Version",
			"Book",
			"Chapter",
			"Verse",
			"Text",
		}
		searchResult, err := s.Index.Search(searchRequest)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("Error while executing search query.")

			if err := setResponseText(resp, "Bible Scholar is experiencing internal errors. Please try again later.", "Processing Error"); err != nil {
				log.WithFields(log.Fields{
					"err": err,
				}).Error("Error while creating response body.")
				c.JSON(500, resp.Data())
				return
			}

			c.JSON(500, resp.Data())
			return
		}

		// https://godoc.org/github.com/blevesearch/bleve#SearchResult
		if len(searchResult.Status.Errors) != 0 {
			log.WithFields(log.Fields{
				"errors": searchResult.Status.Errors,
				"index":  s.Index.Name(),
				"query":  query,
			}).Warn("Encountered non-fatal errors when fetching query result.")
		}

		if searchResult.Hits.Len() < 1 {
			log.WithFields(log.Fields{
				"nhits": searchResult.Hits.Len(),
				"index": s.Index.Name(),
				"query": query,
			}).Warn("Did not find any matching results.")
			if err := setResponseText(
				resp,
				"We didn't find any verses matching that phrase. Try a shorter phrase, or try rewording your search phrase.",
				"No results found",
			); err != nil {
				log.WithFields(log.Fields{
					"err": err,
				}).Error("Error while creating response body.")
				c.JSON(500, resp.Data())
				return
			}
			c.JSON(200, resp.Data())
			return
		}

		// FIXME: Separate responses for card body and for voice response
		// https://godoc.org/github.com/blevesearch/bleve/search#DocumentMatch
		resultObject := searchResult.Hits[0].Fields
		log.WithFields(log.Fields{
			"nhits": searchResult.Hits.Len(),
			"index": s.Index.Name(),
			"query": query,
		}).Warn("Found matching results.")
		if err = setResponseText(
			resp,
			fmt.Sprintf("Best match is from %s chapter %d verse %d from the %s translation. %s",
				resultObject["Book"],
				int(resultObject["Chapter"].(float64)),
				int(resultObject["Verse"].(float64)),
				resultObject["Version"],
				resultObject["Text"],
			),
			fmt.Sprintf("Found match: %s %d:%d (%s)",
				resultObject["Book"],
				int(resultObject["Chapter"].(float64)),
				int(resultObject["Verse"].(float64)),
				resultObject["Version"],
			),
		); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("Error while creating response body.")
			c.JSON(500, resp.Data())
			return
		}
		c.JSON(200, resp.Data())
		return
	}
}
