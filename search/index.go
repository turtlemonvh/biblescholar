package biblescholar

import (
	"encoding/csv"
	"fmt"
	"github.com/blevesearch/bleve"
	"os"
	"io"
	"path/filepath"
	"unicode/utf8"
)

const indexName = "verses.bleve"

func CreateOrOpenIndex() bleve.Index {
	var index bleve.Index
	if _, err := os.Stat(indexName); os.IsNotExist(err) {
		mapping := bleve.NewIndexMapping()
		index, _ = bleve.New(indexName, mapping)
	} else {
		index, _ = bleve.Open(indexName)
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
		for {
			record, err := r.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nindexed, err
			}

			verse := NewVerseFromLine(record)
			index.Index(verse.Id(), verse)

			nindexedPerFile++
			nindexed++

			if nindexed%10 == 0 {
				fmt.Printf("Indexed %d records from: %s [ %d total ] \n", nindexedPerFile, match, nindexed)
			}
		}
	}

	return nindexed, nil
}
