// Copyright © 2017 Ott-Consult UG (haftungsbeschränkt), Jörn Ott <go@ott-consult.de>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/joernott/go-xymon-remotemonitor/monitor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var hostDir string
var xymonServer string
var xymonPort int
var xymonTimeout time.Duration
var logLevel int
var logFile string
var dryrun bool

// This represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "go-xymon-remotemonitor",
	Short: "A monitor to check hosts and http(s) URLs.",
	Long:  `This monitor can run on a remote server and monitor various URLs for various machines/hosts.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctrl, err := monitor.NewController(hostDir, xymonServer, xymonPort, xymonTimeout, logLevel, logFile)
		if err != nil {
			os.Exit(1)
		}
		err = ctrl.Run(dryrun)
		if err != nil {
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is etc/xymon-client/remotemonitor/config.yaml)")
	RootCmd.PersistentFlags().StringVarP(&hostDir, "hostdir", "d", "/etc/xymon-client/remotemonitor/hosts.d/", "Where to find the host definitions")
	RootCmd.PersistentFlags().StringVarP(&xymonServer, "server", "s", "", "Hostname or IP of thre XYMon server")
	RootCmd.PersistentFlags().IntVarP(&xymonPort, "port", "p", 1984, "The port, xymon is listening on (defaults to 1984)")
	RootCmd.PersistentFlags().DurationVarP(&xymonTimeout, "timeout", "t", time.Second*3, "Timeout duration for the connext to the XYMon server (default 3s)")
	RootCmd.PersistentFlags().IntVarP(&logLevel, "loglevel", "v", 4, "Log Level between 1 (Panic) and 6 (Debug), defaults to 4 (Warn)")
	RootCmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", "", "Where to log to, defaults to stdout")
	RootCmd.PersistentFlags().BoolVar(&dryrun, "dryrun", false, "Do a dry run, log but don't report to XYMon")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName("config")                          // name of config file (without extension)
	viper.AddConfigPath("/etc/xymon-client/remotemonitor") // adding home directory as first search path
	viper.AutomaticEnv()                                   // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		//fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println(err)
	}
}
