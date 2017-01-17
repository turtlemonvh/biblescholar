package server

import (
	"fmt"
	"net/http"

	"github.com/Jeffail/gabs"
	log "github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/gin-gonic/gin"
)

const (
	// Types
	intentRequest       string = "IntentRequest"
	launchRequest       string = "LaunchRequest"
	sessionEndedRequest string = "SessionEndedRequest"
	// Object paths
	requestTypePath       string = "request.type"
	requestIntentNamePath string = "request.intent.name"
	requestQueryPath      string = "request.intent.slots.QueryPhrase.value"
)

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

func setResponseText(ro *gabs.Container, txt string, title string, reprompt bool) error {
	responsePath := "outputSpeech"
	if reprompt {
		responsePath = "reprompt.outputSpeech"
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

// Grab the request type
// Should be intentRequest or launchRequest
func getRequestType(req *gabs.Container) (string, error) {
	if !req.ExistsP(requestTypePath) {
		return "", fmt.Errorf("No value at required path '%s'", requestTypePath)
	}
	value, ok := req.Path(requestTypePath).Data().(string)
	if !ok {
		return "", fmt.Errorf("Invalid value supplied for '%s' parameter of request object: %v", requestTypePath, req.Path("type").Data())
	}

	log.WithFields(log.Fields{
		"type": value,
	}).Debug("Identified request type")
	return value, nil
}

// https://developer.amazon.com/public/solutions/alexa/alexa-skills-kit/docs/alexa-skills-kit-interface-reference#response-object
func (s *ServerConfig) getNewResponseTemplate() *gabs.Container {
	resp := gabs.New()
	resp.SetP(s.VersionString(), "version")
	resp.SetP(map[string]interface{}{}, "sessionAttributes")

	// Eventually if they want more results and keep the session open
	resp.SetP(true, "response.shouldEndSession")

	// Remove unused
	resp.DeleteP("response.outputSpeech")
	resp.DeleteP("response.reprompt")
	resp.DeleteP("response.directives")
	resp.DeleteP("response.card.image")
	resp.DeleteP("response.card.text")

	return resp
}

// Prompt in the case of no set slots
// Keeps the session open for easier re-prompting
func (s *ServerConfig) setPromptResponse(c *gin.Context, req *gabs.Container, resp *gabs.Container) {
	// Keep session open
	resp.SetP(false, "response.shouldEndSession")

	// Set both the main response object and the prompt
	for _, op := range []bool{true, false} {
		if err := setResponseText(
			resp,
			"Ask BibleScholar to 'search for' or 'lookup' a phrase.",
			"BibleScholar Help",
			op,
		); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("Error while creating response body.")
			c.JSON(http.StatusInternalServerError, resp.Data())
			return
		}
	}

	c.JSON(http.StatusOK, resp.Data())
}

// Handlers

// Handle a general start request with no context
func (s *ServerConfig) handleLaunchRequest(c *gin.Context, req *gabs.Container, resp *gabs.Container) {
	s.setPromptResponse(c, req, resp)
}

// Handle a request to end the session
// https://developer.amazon.com/public/solutions/alexa/alexa-skills-kit/docs/custom-standard-request-types-reference#sessionendedrequest
// Basically a no-op response with logging
func (s *ServerConfig) handleSessionEndedRequest(c *gin.Context, req *gabs.Container, resp *gabs.Container) {
	log.WithFields(log.Fields{
		"reason": req.Path("reason"),
	}).Info("Session ended")
	c.JSON(http.StatusOK, gin.H{
		"response": map[string]interface{}{
			"shouldEndSession": true,
		},
	})
}

// Completely handle intent requests
func (s *ServerConfig) handleIntentRequest(c *gin.Context, req *gabs.Container, resp *gabs.Container) {
	// All searches end after 1 request
	resp.SetP(true, "response.shouldEndSession")

	// Check request has data where we expect
	if !req.ExistsP(requestIntentNamePath) {
		log.WithFields(log.Fields{
			"path": requestIntentNamePath,
		}).Error("Required path in request object does not exist")
		setResponseText(resp, "Your request is malformed.", "Processing Error", false)
		c.JSON(http.StatusBadRequest, resp.Data())
		return
	}

	intent, ok := req.Path(requestIntentNamePath).Data().(string)

	// https://developer.amazon.com/public/solutions/alexa/alexa-skills-kit/docs/alexa-skills-kit-voice-interface-and-user-experience-testing?ref_=pe_679090_102923190#stopping-and-canceling
	if intent == "AMAZON.StopIntent" || intent == "AMAZON.CancelIntent" {
		s.handleSessionEndedRequest(c, req, resp)
		return
	}

	if !ok || intent != "SearchBible" {
		// We only handle this one intent
		log.WithFields(log.Fields{
			"intentReceived": intent,
			"intentExpected": "SearchBible",
		}).Error("Invalid task intent")
		setResponseText(resp, "Your request is malformed.", "Processing Error", false)
		c.JSON(http.StatusBadRequest, resp.Data())
		return
	}

	if !req.ExistsP(requestQueryPath) {
		// This is a SearchBible request without a query
		s.setPromptResponse(c, req, resp)
		return
	}

	queryText, ok := req.Path(requestQueryPath).Data().(string)
	if !ok {
		log.WithFields(log.Fields{
			"queryValue": requestQueryPath,
		}).Error("Invalid query value. Expected string.")
		setResponseText(resp, "Your request is malformed.", "Processing Error", false)
		c.JSON(http.StatusBadRequest, resp.Data())
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
		setResponseText(resp, "Bible Scholar is experiencing internal errors. Please try again later.", "Processing Error", false)
		c.JSON(http.StatusInternalServerError, resp.Data())
		return
	}

	// https://godoc.org/github.com/blevesearch/bleve#SearchResult
	if len(searchResult.Status.Errors) != 0 {
		log.WithFields(log.Fields{
			"errors": searchResult.Status.Errors,
			"index":  s.Index.Name(),
			"query":  queryText,
		}).Warn("Encountered non-fatal errors when fetching query result.")
	}

	// Not found
	if searchResult.Hits.Len() < 1 {
		log.WithFields(log.Fields{
			"nhits": searchResult.Hits.Len(),
			"index": s.Index.Name(),
			"query": queryText,
		}).Warn("Did not find any matching results.")
		if err := setResponseText(
			resp,
			"We didn't find any verses matching that phrase. Try a shorter phrase, or try rewording your search phrase.",
			"No results found",
			false,
		); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("Error while creating response body.")
			c.JSON(http.StatusInternalServerError, resp.Data())
			return
		}
		c.JSON(http.StatusOK, resp.Data())
		return
	}

	// FIXME: Separate responses for card body and for voice response
	// https://godoc.org/github.com/blevesearch/bleve/search#DocumentMatch
	resultObject := searchResult.Hits[0].Fields
	log.WithFields(log.Fields{
		"nhits": searchResult.Hits.Len(),
		"index": s.Index.Name(),
		"query": queryText,
	}).Info("Found matching results.")
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
		false,
	); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Error while creating response body.")
		c.JSON(http.StatusInternalServerError, resp.Data())
		return
	}
	c.JSON(http.StatusOK, resp.Data())
}
