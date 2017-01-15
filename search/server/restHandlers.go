package server

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// Handle a general search query via q query param
func searchHandler(s *ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Query, required
		q, exists := c.GetQuery("q")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"err": fmt.Sprintf("Missing required query parameter 'q'"),
			})
			return
		}

		// Num results, optional
		size, exists := c.GetQuery("size")
		if !exists {
			size = "10"
		}
		isize, err := strconv.Atoi(size)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err": fmt.Sprintf("Invalid format of query parameter 'size', expected int, got: %v", size),
			})
			return
		}

		// From, optional
		from, exists := c.GetQuery("from")
		if !exists {
			from = "0"
		}
		ifrom, err := strconv.Atoi(from)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err": fmt.Sprintf("Invalid format of query parameter 'from', expected int, got: %v", from),
			})
			return
		}

		query := bleve.NewQueryStringQuery(q)
		searchRequest := bleve.NewSearchRequestOptions(query, 10, 0, false)
		searchRequest.Fields = []string{
			"Version",
			"Book",
			"Chapter",
			"Verse",
			"Text",
		}
		searchRequest.Size = isize
		searchRequest.From = ifrom
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
			"q":        q,
			"size":     isize,
			"from":     ifrom,
			"nresults": len(searchResult.Hits),
		}).Info("Search status")
		c.JSON(http.StatusOK, searchResult)

		return
	}
}
