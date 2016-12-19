package bblsearch

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
}

func (v *Verse) Id() string {
	return fmt.Sprintf("%s-%d-%d-%s", v.Book, v.Chapter, v.Verse, v.Version)
}

// From a line in a tsv/csv
func NewVerseFromLine(line []string) Verse {
	chapter, _ := strconv.Atoi(line[2])
	verse, _ := strconv.Atoi(line[3])

	return Verse{
		Version: line[0],
		Book:    line[1],
		Chapter: chapter,
		Verse:   verse,
		Text:    line[4],
	}
}
