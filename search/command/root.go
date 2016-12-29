package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/turtlemonvh/biblescholar"
	"github.com/turtlemonvh/biblescholar/server"
	"os"
)

/*
Project is called `biblescholar`. The command line interface is called `bblsearch`.
*/

var searchCmdV *cobra.Command
var buildCommit string = "UNKNOWN"
var buildBranch string = "UNKNOWN"

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
	RootCmd.AddCommand(serverCmd)
	searchCmdV = RootCmd

	// FIXME: Add support for multiple outputs and handling log levels via command line or env variable
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	RootCmd.PersistentFlags().StringP(
		"index-path", "i", biblescholar.DefaultIndexName,
		fmt.Sprintf("path to bleve index. Default is: %s", biblescholar.DefaultIndexName),
	)
	indexCmd.Flags().StringP("data-dir", "d", "downloads", "directory containing tsv data files to use in indexing")
	serverCmd.Flags().IntP("port", "p", 8000, "port to run server on")
}

var RootCmd = &cobra.Command{
	Use:   "bblsearch",
	Short: "bblsearch is a search interface for the Bible",
	Run: func(cmd *cobra.Command, args []string) {
		InitializeConfig()
		fmt.Printf("bblsearch %s (%s)\n", buildBranch, buildCommit)
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
		viper.BindPFlag("index-path", cmd.Flags().Lookup("index-path"))

		index := biblescholar.CreateOrOpenIndex(viper.GetString("index-path"))
		fmt.Println("Adding content to: ", index)

		_, err := biblescholar.IndexFromTSVs(index, viper.GetString("data-dir"))
		if err != nil {
			log.Fatal(err)
		}
	},
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a server that fields responses to queries from alexa",
	Long:  `Start a server that fields responses to queries from alexa`,
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("port", cmd.Flags().Lookup("port"))
		viper.BindPFlag("index-path", cmd.Flags().Lookup("index-path"))

		svr := server.ServerConfig{
			Port:        viper.GetInt("port"),
			BuildCommit: buildCommit,
			BuildBranch: buildBranch,
			Index:       biblescholar.CreateOrOpenIndex(viper.GetString("index-path")),
		}
		svr.StartServer()
	},
}
