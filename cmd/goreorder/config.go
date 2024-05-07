package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func initializeViper(c *cobra.Command, args ...string) error {
	v := viper.New()
	v.SetConfigName(".goreorder")
	v.SetConfigType("yaml")

	v.AddConfigPath(".")
	// also, add the projet root
	for _, arg := range args {
		stat, err := os.Stat(arg)
		if err != nil {
			continue
		}
		if stat.IsDir() {
			v.AddConfigPath(arg)
		} else {
			v.AddConfigPath(filepath.Dir(arg))
		}
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	v.SetEnvPrefix("GOREORDER")
	v.AutomaticEnv()
	bindFlags(c, v)
	return nil
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		name := f.Name
		if !f.Changed && v.IsSet(name) {
			// ensure that the value is with the correct type
			switch f.Value.Type() {
			case "stringSlice":
				cmd.Flags().Lookup(name).Value.Set(strings.Join(v.GetStringSlice(name), ","))
			default:
				val := v.GetString(name)
				cmd.Flags().Set(name, fmt.Sprintf("%v", val))
			}
		}
	})
}

func printConfigFile(config *ReorderConfig) error {
	// for all flags, get the current value and set it to conf
	enc := yaml.NewEncoder(defaultOutpout)
	enc.SetIndent(2)
	return enc.Encode(&config)
}
