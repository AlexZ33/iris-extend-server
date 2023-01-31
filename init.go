package iris_extend_server

import (
	"crypto/rand"
	"crypto/rsa"
	"flag"
	helper "github.com/AlexZ33/iris-extend-helper"
	"github.com/allegro/bigcache"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/memstore"
	"github.com/pelletier/go-toml"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	Dir          string
	Env          string
	Name         string
	Version      string
	MasterAddr   string
	ProvidesAuth bool
	MaintainerId string
	StartTime    time.Time
	IrisConfig   iris.Configuration
	Config       *toml.Tree
	Path         string
	Filename     string
	Cache        *bigcache.BigCache
	Record       *bigcache.BigCache
	PublicKey    *rsa.PublicKey
	PrivateKey   *rsa.PrivateKey
	Store        *memstore.Store
)

func init() {
	if dir, err := os.Getwd(); err != nil {
		log.Println(err)
	} else {
		Dir = filepath.ToSlash(dir)
	}

	IrisConfig = iris.TOML("./iris.toml")
	flag.StringVar(&Env, "env", "", "set server environment")
	flag.Parse()
	if Env == "" {
		if env, ok := IrisConfig.Other["ServerEnvironment"]; ok {
			Env = env.(string)
		} else {
			Env = "local"
		}
	}
	Path = getFilePath("")
	if config, ok := Configure(Path); ok {
		Config = config
	} else {
		log.Fatalln("fail to load config file")
	}

	Name = helper.GetString(Config, "name")
	Version = helper.GetString(Config, "version")
	MasterAddr = Addr(helper.GetTree(Config, "master"))
	MaintainerId = helper.GetString(Config, "maintainer-id")
	StartTime = time.Now()

	access := helper.GetTree(Config, "access")
	ProvidesAuth = !helper.GetBool(access, "use-external-api")

	cache := helper.GetTree(Config, "cache")
	cacheCfg := bigcache.DefaultConfig(helper.GetDuration(cache, "max-age", time.Minute))
	cacheCfg.CleanWindow = helper.GetDuration(cache, "cleanup-interval", time.Second)
	cacheCfg.HardMaxCacheSize = helper.ParseMegabytes(cache.Get("max-cache-size"))
	if cache, err := bigcache.NewBigCache(cacheCfg); err != nil {
		log.Println(err)
	} else {
		Cache = cache
	}

	recordCfg := bigcache.DefaultConfig(time.Hour)
	if record, err := bigcache.NewBigCache(recordCfg); err != nil {
		log.Println(err)
	} else {
		Record = record
	}

	keySize := helper.GetInt(Config, "crypto.rsa-key-size", 2048)
	if privkey, err := rsa.GenerateKey(rand.Reader, keySize); err != nil {
		log.Println(err)
	} else {
		PrivateKey = privkey
		if pubkey, ok := privkey.Public().(*rsa.PublicKey); ok {
			PublicKey = pubkey
		}
	}
	Store = new(memstore.Store)
}

func getFilePath(filename string) string {
	if filename != "" {
		Filename = filename
	} else {
		Filename = "./config/config." + Env + ".toml"
	}
	return Filename
}
