package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"gopkg.in/olivere/elastic.v3"

	log "github.com/cihub/seelog"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/jpillora/go-ogle-analytics"
	httpr "github.com/julienschmidt/httprouter"
	"github.com/ok-borg/api/conf"
	"github.com/ok-borg/api/endpoints"
	"github.com/ok-borg/api/sitemap"
	"github.com/ok-borg/api/v"
	"github.com/ok-borg/api/v/v1"
	"github.com/ok-borg/api/v/v2"
	"github.com/rs/cors"
	"golang.org/x/oauth2"
)

const (
	githubTokenURL = "https://github.com/login/oauth/access_token"
)

var (
	esAddr             = flag.String("esaddr", "127.0.0.1:9200", "Elastic Search address")
	githubClientId     = flag.String("github-client-id", "", "Github oauth client id")
	githubClientSecret = flag.String("github-client-secret", "", "Github client secret")
	sm                 = flag.String("sitemap", "", "Sitemap location. Leave empty if you don't want a sitemap to be generated")
	analytics          = flag.String("analytics", "", "Analytics tracking id")
	sqlAddr            = flag.String("sqladdr", "127.0.0.1:3306", "Mysql address")
	sqlIds             = flag.String("sqlids", "root:root", "Mysql identifier")
)

var (
	client          *elastic.Client
	analyticsClient *ga.Client
	ep              *endpoints.Endpoints
	db              *gorm.DB
)

func initWithConfFile() {
	// let's assume the conf file is in the same place than the benary.
	c, err := ioutil.ReadFile(".borg.conf.json")
	if err != nil {
		// just return, there is probably not configuration file
		return
	}

	var conf conf.Conf
	if err := json.Unmarshal(c, &conf); err != nil {
		panic(fmt.Sprintf("[initWithConfFile] invalid config format: %s", err.Error()))
	}

	if conf.EsAddr != "" {
		*esAddr = conf.EsAddr
	}
	if conf.Github.ClientId != "" {
		*githubClientId = conf.Github.ClientId
	}
	if conf.Github.ClientSecret != "" {
		*githubClientSecret = conf.Github.ClientSecret
	}
	if conf.Sitemap != "" {
		*sm = conf.Sitemap
	}
	if conf.Analytics != "" {
		*analytics = conf.Analytics
	}
	if conf.Mysql.Addr != "" {
		*sqlAddr = conf.Mysql.Addr
	}
	if conf.Mysql.Ids != "" {
		*sqlIds = conf.Mysql.Ids
	}
}

func init() {
	// read config file before if it exists, so we can replaces the var that was set with the cmdline
	// the cmdline is allowed to overwrite the config file.
	initWithConfFile()
	flag.Parse()

	cl, err := elastic.NewClient(elastic.SetSniff(false), elastic.SetURL(fmt.Sprintf("http://%v", *esAddr)))
	if err != nil {
		panic(err)
	}
	client = cl
	if len(*analytics) > 0 {
		acl, err := ga.NewClient(*analytics)
		if err != nil {
			log.Errorf("Failed to acquire analytics client id: %v", err)
		}
		analyticsClient = acl
	}
}

func main() {
	oauthCfg := &oauth2.Config{
		ClientID:     *githubClientId,
		ClientSecret: *githubClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: githubTokenURL,
		},
		Scopes: []string{"read:org"},
	}

	// init mysql
	var err error
	dsn := fmt.Sprintf("%s@tcp(%s)/borg?parseTime=True", *sqlIds, *sqlAddr)
	if db, err = gorm.Open("mysql", dsn); err != nil {
		panic(fmt.Sprintf("[init] unable to initialize gorm: %s", err.Error()))
	}
	defer db.Close()

	ep = endpoints.NewEndpoints(oauthCfg, client, analyticsClient, db)
	r := httpr.New()
	if len(*sm) > 0 {
		go sitemapLoop(*sm, client)
	}

	// decl routes
	common.Init(client, analyticsClient, ep, db, *githubClientId)
	v1.Init(r, client, analyticsClient, ep, db)
	v2.Init(r, client, analyticsClient, ep, db)

	handler := cors.New(cors.Options{AllowedHeaders: []string{"*"}, AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"}}).Handler(r)
	log.Info("Starting http server")
	log.Critical(http.ListenAndServe(fmt.Sprintf(":%v", 9992), handler))
}

func sitemapLoop(path string, client *elastic.Client) {
	first := true
	for {
		if !first {
			time.Sleep(30 * time.Minute)
		}
		first = false
		sitemap.GenerateSitemap(path, client)
	}
}
