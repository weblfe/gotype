// Package cmd /*
package cmd

import (
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/weblfe/gotype/run"
	"os"
)

var (
	cfgFile string
	bin     string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gotype",
	Short: "Displays the type of the specified command",
	Long:  `Using the type command,you can view the type of a specified command and determine whether the command is an internal command or an external command.`,
	// Uncomment the following line if your bare application
	// has an action associated with it: args 是除 command options 以外的命令的不定参数
	Run: func(cmd *cobra.Command, args []string) {
		var (
			runner = run.NewRunner(os.Stdin, os.Stderr, os.Stdout).Bind(bin)
		)
		if v, err := cmd.Flags().GetString(`path`); err == nil && v != "" {
			runner.Exec(`path`, v)
			return
		}
		if v, err := cmd.Flags().GetString(`all`); err == nil && v != "" {
			runner.Exec(`all`, v)
			return
		}
		if v, err := cmd.Flags().GetString(`type`); err == nil && v != "" {
			runner.Exec(`type`, v)
			return
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gotype.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("type", "t", ``, `Output "file", "alias", or "builtin" to indicate that the given instruction is "external instruction", "command alias", or "internal instruction", respectively`)
	rootCmd.Flags().StringP("path", "p", ``, `If the given instruction is an external instruction, its absolute path is displayed.`)
	rootCmd.Flags().StringP("all", "a", ``, `Displays information about the given command, including the command alias, in the PATH specified by the environment variable "PATH".`)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".type" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".gotype")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		//fmt.Println("Using config file:", viper.ConfigFileUsed())
		bin = viper.GetString(`builtin_type_bin`)
	} else {
		bin = os.Getenv(`BUILTIN_TYPE_BIN`)
	}
}
