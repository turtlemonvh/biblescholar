package biblescholar

import (
	"fmt"
	"strconv"
)

type Verse struct {
	Book    string
	Chapter int
	Verse   int
	Text    string
	Version string
	// Create a string that includes version and book
	// We save this field because we can't do nested aggregations, but we do want to get book per version for visualizations
	VersionBook string
}

func (v *Verse) Id() string {
	return fmt.Sprintf("%s-%d-%d-%s", v.Book, v.Chapter, v.Verse, v.Version)
}

// Implementation so that items of this type are bound to the correct mapping
// https://godoc.org/github.com/blevesearch/bleve/mapping#Classifier
// https://github.com/blevesearch/bleve/blob/v0.5.0/index.go#L87
func (v *Verse) Type() string {
	return "verse"
}

// From a line in a tsv/csv
func NewVerseFromLine(line []string) *Verse {
	chapter, _ := strconv.Atoi(line[2])
	verse, _ := strconv.Atoi(line[3])

	return &Verse{
		Version:     line[0],
		Book:        line[1],
		Chapter:     chapter,
		Verse:       verse,
		Text:        line[4],
		VersionBook: fmt.Sprintf("%s-%s", line[0], line[1]),
	}
}
