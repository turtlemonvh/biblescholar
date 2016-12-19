package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/turtlemonvh/biblescholar"
	"os"
)

/*
Project is called `biblescholar`. The command line interface is called `bblsearch`.
*/

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

		index := biblescholar.CreateOrOpenIndex()
		fmt.Println("Adding content to: ", index)

		_, err := biblescholar.IndexFromTSVs(index, viper.GetString("data-dir"))
		if err != nil {
			log.Fatal(err)
		}
	},
}
