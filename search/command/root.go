package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/turtlemonvh/bblsearch"
	"os"
)

var searchCmdV *cobra.Command

func main() {
	RootCmd.Execute()
}

func InitializeConfig() {
	viper.SetConfigName("bblsearch")
	viper.AddConfigPath("/etc/bblsearch/")
	viper.AddConfigPath("$HOME/.bblsearch")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("BBLSEARCH_")
	viper.AutomaticEnv()
}

func init() {
	RootCmd.AddCommand(indexCmd)
	searchCmdV = RootCmd

	// FIXME: Add support for multiple outputs and handling log levels via command line or env variable
	// https://golang.org/src/io/multi.go?s=1355:1397#L47
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)

	indexCmd.Flags().StringP("data-dir", "d", "downloads", "directory containing tsv data files to use in indexing")
}

var RootCmd = &cobra.Command{
	Use:   "bblsearch",
	Short: "bblsearch is a search interface for the Bible",
	Run: func(cmd *cobra.Command, args []string) {
		InitializeConfig()
		fmt.Println("bblsearch v0.1")
	},
}

var indexLongDesc = `Searches data-dir for tsv files to add to index.  Files should look like:

<version>  <Book>  <Chapter #>  <Verse #>  <Verse Text>
`
var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index from a collection of tsv files",
	Long:  indexLongDesc,
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("data-dir", cmd.Flags().Lookup("data-dir"))

		index := bblsearch.CreateOrOpenIndex()
		fmt.Println("Adding content to: ", index)

		verses, _ := bblsearch.VersesFromTSVs(viper.GetString("data-dir"))
		for verseIndex, verse := range verses {
			if verseIndex%10 == 0 {
				fmt.Printf("Indexed %d of %d [ %3.2f %% ] \n", verseIndex, len(verses), float64(verseIndex)/float64(len(verses)))
			}

			index.Index(verse.Id(), verse)
		}
	},
}
