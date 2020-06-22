package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type DbConnectStruct struct {
	Type string
	Host string
	Port string
	User string
	DB   string
	Pass string
}

type DbCheckStruct struct {
	CheckType string
	Interval  string
	Src       string
	Dst       string
	Query     struct {
		Src     string
		Dst     string
		Srcfile string
		Dstfile string
	}
}

type DatamonConfig struct {
	DB     map[string]DbConnectStruct
	Checks map[string]DbCheckStruct
}

func (check DbCheckStruct) QueryStrings() (srcQuery string, dstQuery string) {
	if check.Query.Srcfile != "" {
		srcQuery = stringSQL(check.Query.Srcfile)
	} else {
		srcQuery = check.Query.Src
	}
	if check.Query.Dstfile != "" {
		dstQuery = stringSQL(check.Query.Dstfile)
	} else {
		dstQuery = check.Query.Dst
	}
	return
}

//ReadConfig read app config file
func ReadConfig() (datamonConf DatamonConfig) {
	conf := viper.New()
	conf.SetEnvPrefix("datamon")
	conf.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	conf.AutomaticEnv()
	fmt.Println("Reading config")
	conf.SetConfigName("config")
	conf.SetConfigType("yaml")
	conf.AddConfigPath(".")
	conf.AddConfigPath("/etc/datamon/")
	conf.AddConfigPath(os.Getenv("CONFIG_PATH"))
	conf.WatchConfig()
	conf.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Config file changed:", e.Name)
		os.Exit(0)
	})
	err := conf.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	err = conf.Unmarshal(&datamonConf)
	if err != nil {
		log.Fatalf("Unable to decode datamon config, %v", err)
	}
	return
}

func stringSQL(file string) string {
	content, err := ioutil.ReadFile(fmt.Sprintf("files/%v", file))
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}
