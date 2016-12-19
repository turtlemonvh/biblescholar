package bblsearch

import (
	"encoding/csv"
	"fmt"
	"github.com/blevesearch/bleve"
	"os"
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

func VersesFromTSVs(dirpath string) ([]Verse, error) {
	var verses []Verse
	matches, err := filepath.Glob(fmt.Sprintf("%s/*.tsv", dirpath))
	if err != nil {
		return verses, err
	}

	for _, match := range matches {
		fmt.Println("Records from: ", match)

		f, err := os.Open(match)
		if err != nil {
			return verses, err
		}

		r := csv.NewReader(f)

		tabRune, _ := utf8.DecodeRuneInString("\t")
		r.Comma = tabRune
		records, err := r.ReadAll()
		if err != nil {
			return verses, err
		}

		for _, record := range records {
			verses = append(verses, NewVerseFromLine(record))
		}

		fmt.Printf("Found %d records from: %s\n", len(records), match)
	}

	return verses, nil
}
