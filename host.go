package iris_extend_server

import (
	helper "github.com/AlexZ33/iris-extend-helper"
	jsoniter "github.com/json-iterator/go"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/host"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func Serve(app *iris.Application) {
	app.Configure(iris.WithConfiguration(IrisConfig))
	if err := app.Build(); err != nil {
		app.Logger().Fatal(err)
	}

	if hosts, ok := Config.Get("worker").([]*toml.Tree); ok {
		workers := make([]*http.Server, 0)
		for _, host := range hosts {
			addr := Addr(host)
			workers = append(workers, &http.Server{Addr: addr})
		}
		for _, worker := range workers {
			go app.NewHost(worker).ListenAndServe()
		}
	}

	app.Run(iris.Addr(MasterAddr, func(su *host.Supervisor) {
		su.RegisterOnShutdown(func() {
			log.Println("master server terminated")
		})
	}))
}

func Addr(server *toml.Tree) string {
	host := helper.GetString(server, "host")
	port := helper.GetString(server, "port", "8080")
	if host == "" && Env == "local" {
		host = helper.GetString(Config, "server-host", "localhost")
	}
	return host + ":" + port
}

func Configure(filename string) (*toml.Tree, bool) {
	// filename : config.local.toml/ config.prod.toml
	path := filename
	config, err := toml.LoadFile(path)
	if err != nil {
		log.Println(err)
		return nil, false
	} else if config.Has("config-file-url") {
		url := helper.GetString(config, "config-file-url")
		res, err := http.Get(url)
		if err != nil {
			log.Println(err)
		} else {
			defer res.Body.Close()
			content, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Println(err)
			} else {
				contentType := res.Header.Get("Content-Type")
				if strings.HasPrefix(contentType, "application/json") {
					object := map[string]interface{}{}
					err := jsoniter.Unmarshal(content, &object)
					if err != nil {
						log.Println(err)
					} else {
						tree, err := toml.TreeFromMap(object)
						if err != nil {
							log.Println(err)
						} else {
							return tree, true
						}
					}
				} else {
					tree, err := toml.LoadBytes(content)
					if err != nil {
						log.Println(err)
					} else {
						return tree, true
					}
				}
			}
		}
	}
	return config, true
}

func NewContext() iris.Context {
	return context.NewContext(iris.Default())
}

func CopyContext(ctx iris.Context) iris.Context {
	newCtx := NewContext()
	ctx.Values().Visit(func(key string, value interface{}) {
		newCtx.Values().Set(key, value)
	})
	return newCtx
}

func IsLocal() bool {
	return Env == "local"
}
