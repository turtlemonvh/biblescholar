package biblescholar

import (
	"encoding/csv"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"
)

const DefaultIndexName = "verses.bleve"

func CreateOrOpenIndex(indexName string) bleve.Index {
	var index bleve.Index
	var err error
	if _, err = os.Stat(indexName); os.IsNotExist(err) {
		idxMapping := bleve.NewIndexMapping()

		// Add field mappings
		textMapping := bleve.NewTextFieldMapping()
		textMapping.Analyzer = "en"
		textMapping.IncludeInAll = true

		bookMapping := bleve.NewTextFieldMapping()
		bookMapping.Analyzer = keyword.Name
		bookMapping.IncludeInAll = false
		bookMapping.IncludeTermVectors = false

		versionMapping := bleve.NewTextFieldMapping()
		versionMapping.Analyzer = keyword.Name
		versionMapping.IncludeInAll = false
		versionMapping.IncludeTermVectors = false

		versionBookMapping := bleve.NewTextFieldMapping()
		versionBookMapping.Analyzer = keyword.Name
		versionBookMapping.IncludeInAll = false
		versionBookMapping.IncludeTermVectors = false

		chapterMapping := bleve.NewNumericFieldMapping()
		chapterMapping.IncludeInAll = false

		verseMapping := bleve.NewNumericFieldMapping()
		verseMapping.IncludeInAll = false

		// Add document mapping, including field mappings
		docMapping := bleve.NewDocumentStaticMapping()
		docMapping.AddFieldMappingsAt("Book", bookMapping)
		docMapping.AddFieldMappingsAt("Version", versionMapping)
		docMapping.AddFieldMappingsAt("VersionBook", versionBookMapping)
		docMapping.AddFieldMappingsAt("Chapter", chapterMapping)
		docMapping.AddFieldMappingsAt("Verse", verseMapping)
		docMapping.AddFieldMappingsAt("Text", textMapping)
		docMapping.Dynamic = false
		idxMapping.AddDocumentMapping("verse", docMapping)

		index, err = bleve.New(indexName, idxMapping)
	} else {
		index, err = bleve.Open(indexName)
	}
	if err != nil {
		log.Fatal(err)
	}
	return index
}

func IndexFromTSVs(index bleve.Index, dirpath string) (int, error) {
	nindexed := 0
	matches, err := filepath.Glob(fmt.Sprintf("%s/*.tsv", dirpath))
	if err != nil {
		return nindexed, err
	}

	for _, match := range matches {
		fmt.Println("Starting to index records from: ", match)
		nindexedPerFile := 0

		f, err := os.Open(match)
		if err != nil {
			return nindexed, err
		}

		r := csv.NewReader(f)

		tabRune, _ := utf8.DecodeRuneInString("\t")
		r.Comma = tabRune

		// FIXME: Read in a line at a time, and feed in directly to indexer
		// FIXME: Use the bulk indexer for performance improvements
		b := index.NewBatch()
		for {
			record, err := r.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nindexed, err
			}

			verse := NewVerseFromLine(record)

			// FIXME: Check for error
			b.Index(verse.Id(), verse)

			nindexedPerFile++
			nindexed++

			if nindexed%100 == 0 {
				err := index.Batch(b)
				if err != nil {
					return nindexed, err
				}
				b = index.NewBatch()
				fmt.Printf("Indexed %d records from: %s [ %d total ] \n", nindexedPerFile, match, nindexed)
			}
		}

		// CLeanup batch
		err = index.Batch(b)
		if err != nil {
			return nindexed, err
		}
		fmt.Printf("Indexed %d records from: %s [ %d total ] \n", nindexedPerFile, match, nindexed)
	}

	return nindexed, nil
}
