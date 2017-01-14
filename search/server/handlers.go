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

// Handle a general start request with no context
func (s *ServerConfig) setPromptResponse(c *gin.Context, req *gabs.Container, resp *gabs.Container) {
	// Keep session open
	resp.SetP(false, "shouldEndSession")
	if err := setResponseText(
		resp,
		"Ask BibleScholar to search for or lookup a phrase.",
		"BibleScholar Help",
		true,
	); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Error while creating response body.")
		c.JSON(http.StatusInternalServerError, resp.Data())
		return
	}
	c.JSON(http.StatusOK, resp.Data())
}

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
	c.JSON(http.StatusOK, gin.H{})
}

// Completely handle intent requests
// FIXME: Handle empty slots with a prompt
func (s *ServerConfig) handleIntentRequest(c *gin.Context, req *gabs.Container, resp *gabs.Container) {
	// All searches end after 1 request
	resp.SetP(true, "shouldEndSession")

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
