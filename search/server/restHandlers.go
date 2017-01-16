package server

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/blevesearch/bleve/search/query"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

var (
	defaultQueryString string = "for God so loved the world"
)

type SearchConfig struct {
	q               string
	size            int64
	from            int64
	shouldHighlight bool
}

func (s *ServerConfig) processQueryParams(c *gin.Context, defaultQuery *string) (*bleve.SearchRequest, error) {
	var nilReq *bleve.SearchRequest

	// Query, required
	q, exists := c.GetQuery("q")
	if !exists {
		if defaultQuery != nil {
			q = *defaultQuery
		} else {
			err := fmt.Errorf("Missing required query parameter 'q'")
			c.JSON(http.StatusBadRequest, gin.H{
				"err": err.Error(),
			})
			return nilReq, err
		}
	}

	// Num results, optional
	size, exists := c.GetQuery("size")
	if !exists {
		size = "10"
	}
	isize, err := strconv.Atoi(size)
	if err != nil {
		err := fmt.Errorf("Invalid format of query parameter 'size', expected int, got: %v", size)
		c.JSON(http.StatusBadRequest, gin.H{
			"err": err.Error(),
		})
		return nilReq, err
	}

	// From, optional
	from, exists := c.GetQuery("from")
	if !exists {
		from = "0"
	}
	ifrom, err := strconv.Atoi(from)
	if err != nil {
		err := fmt.Errorf("Invalid format of query parameter 'from', expected int, got: %v", from)
		c.JSON(http.StatusBadRequest, gin.H{
			"err": err.Error(),
		})
		return nilReq, err
	}

	// highlight?
	_, shouldHighlight := c.GetQuery("highlight")

	query := bleve.NewQueryStringQuery(q)
	searchRequest := bleve.NewSearchRequestOptions(query, 10, 0, false)
	searchRequest.Fields = []string{
		"Version",
		"Book",
		"Chapter",
		"Verse",
		"Text",
	}
	if shouldHighlight {
		searchRequest.Highlight = bleve.NewHighlightWithStyle("html")
	}
	searchRequest.Size = isize
	searchRequest.From = ifrom

	return searchRequest, nil
}

// FIXME: Return HTML results on error
func htmlHandler(s *ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		nHtmlRequests.Inc(1)
		start := time.Now()

		// FIXME: Return http instead
		searchRequest, err := s.processQueryParams(c, &defaultQueryString)
		if err != nil {
			// Exact error is set on resp. object in the process function
			return
		}

		searchResult, err := s.Index.Search(searchRequest)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("Error while executing search query.")
			c.JSON(http.StatusInternalServerError, gin.H{
				"err": err.Error(),
			})
			return
		}

		userQuery := searchRequest.Query.(*query.QueryStringQuery).Query

		log.WithFields(log.Fields{
			"q":               userQuery,
			"size":            searchRequest.Size,
			"from":            searchRequest.From,
			"nresults":        len(searchResult.Hits),
			"shouldHighlight": (searchRequest.Highlight == nil),
		}).Debug("Composed search object")

		dur := time.Since(start)

		// Initialize data for template
		data := struct {
			Title         string
			Headline      string
			Query         string
			Size          int
			ReturnResults bool
			Hits          search.DocumentMatchCollection
		}{
			"BibleScholar query interface",
			fmt.Sprintf(`BibleScholar - Listing %d of %d results for "%s" (%s)`, len(searchResult.Hits), searchResult.Total, userQuery, dur.String()),
			userQuery,
			searchRequest.Size,
			true,
			searchResult.Hits,
		}

		if err := s.template.Execute(c.Writer, data); err != nil {
			log.WithFields(log.Fields{
				"err": err.Error(),
			}).Error("Error executing template")
		}

	}
}

// Handle a general search query via q query param
func searchHandler(s *ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		nRestRequests.Inc(1)

		searchRequest, err := s.processQueryParams(c, nil)
		if err != nil {
			// Exact error is set on resp. object in the process function
			return
		}

		searchResult, err := s.Index.Search(searchRequest)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("Error while executing search query.")
			c.JSON(http.StatusInternalServerError, gin.H{
				"err": err.Error(),
			})
			return
		}

		log.WithFields(log.Fields{
			"q":               searchRequest.Query.(*query.QueryStringQuery).Query,
			"size":            searchRequest.Size,
			"from":            searchRequest.From,
			"nresults":        len(searchResult.Hits),
			"shouldHighlight": (searchRequest.Highlight == nil),
		}).Debug("Composed search object")

		c.JSON(http.StatusOK, searchResult)

		return
	}
}
