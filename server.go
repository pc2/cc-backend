package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/ClusterCockpit/cc-jobarchive/api"
	"github.com/ClusterCockpit/cc-jobarchive/auth"
	"github.com/ClusterCockpit/cc-jobarchive/config"
	"github.com/ClusterCockpit/cc-jobarchive/graph"
	"github.com/ClusterCockpit/cc-jobarchive/graph/generated"
	"github.com/ClusterCockpit/cc-jobarchive/metricdata"
	"github.com/ClusterCockpit/cc-jobarchive/templates"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var db *sqlx.DB

// Format of the configurartion (file). See below for the defaults.
type ProgramConfig struct {
	// Address where the http (or https) server will listen on (for example: 'localhost:80').
	Addr string `json:"addr"`

	// Disable authentication (for everything: API, Web-UI, ...)
	DisableAuthentication bool `json:"disable-authentication"`

	// Folder where static assets can be found, will be served directly
	StaticFiles string `json:"static-files"`

	// Currently only SQLite3 ist supported, so this should be a filename
	DB string `json:"db"`

	// Path to the job-archive
	JobArchive string `json:"job-archive"`

	// Make the /api/jobs/stop_job endpoint do the heavy work in the background.
	AsyncArchiving bool `json:"async-archive"`

	// Keep all metric data in the metric data repositories,
	// do not write to the job-archive.
	DisableArchive bool `json:"disable-archive"`

	// For LDAP Authentication and user syncronisation.
	LdapConfig *auth.LdapConfig `json:"ldap"`

	// If both those options are not empty, use HTTPS using those certificates.
	HttpsCertFile string `json:"https-cert-file"`
	HttpsKeyFile  string `json:"https-key-file"`

	// If overwriten, at least all the options in the defaults below must
	// be provided! Most options here can be overwritten by the user.
	UiDefaults map[string]interface{} `json:"ui-defaults"`
}

var programConfig ProgramConfig = ProgramConfig{
	Addr:                  "0.0.0.0:8080",
	DisableAuthentication: false,
	StaticFiles:           "./frontend/public",
	DB:                    "./var/job.db",
	JobArchive:            "./var/job-archive",
	AsyncArchiving:        true,
	DisableArchive:        false,
	LdapConfig: &auth.LdapConfig{
		Url:        "ldap://localhost",
		UserBase:   "ou=hpc,dc=rrze,dc=uni-erlangen,dc=de",
		SearchDN:   "cn=admin,dc=rrze,dc=uni-erlangen,dc=de",
		UserBind:   "uid={username},ou=hpc,dc=rrze,dc=uni-erlangen,dc=de",
		UserFilter: "(&(objectclass=posixAccount)(uid=*))",
	},
	HttpsCertFile: "",
	HttpsKeyFile:  "",
	UiDefaults: map[string]interface{}{
		"analysis_view_histogramMetrics":     []string{"flops_any", "mem_bw", "mem_used"},
		"analysis_view_scatterPlotMetrics":   [][]string{{"flops_any", "mem_bw"}, {"flops_any", "cpu_load"}, {"cpu_load", "mem_bw"}},
		"job_view_nodestats_selectedMetrics": []string{"flops_any", "mem_bw", "mem_used"},
		"job_view_polarPlotMetrics":          []string{"flops_any", "mem_bw", "mem_used", "net_bw", "file_bw"},
		"job_view_selectedMetrics":           []string{"flops_any", "mem_bw", "mem_used"},
		"plot_general_colorBackground":       true,
		"plot_general_colorscheme":           []string{"#00bfff", "#0000ff", "#ff00ff", "#ff0000", "#ff8000", "#ffff00", "#80ff00"},
		"plot_general_lineWidth":             1,
		"plot_list_jobsPerPage":              10,
		"plot_list_selectedMetrics":          []string{"cpu_load", "mem_used", "flops_any", "mem_bw", "clock"},
		"plot_view_plotsPerRow":              4,
		"plot_view_showPolarplot":            true,
		"plot_view_showRoofline":             true,
		"plot_view_showStatTable":            true,
	},
}

func main() {
	var flagReinitDB, flagStopImmediately, flagSyncLDAP bool
	var flagConfigFile string
	var flagNewUser, flagDelUser string
	flag.BoolVar(&flagReinitDB, "init-db", false, "Go through job-archive and re-initialize `job`, `tag`, and `jobtag` tables")
	flag.BoolVar(&flagSyncLDAP, "sync-ldap", false, "Sync the `user` table with ldap")
	flag.BoolVar(&flagStopImmediately, "no-server", false, "Do not start a server, stop right after initialization and argument handling")
	flag.StringVar(&flagConfigFile, "config", "", "Location of the config file for this server (overwrites the defaults)")
	flag.StringVar(&flagNewUser, "add-user", "", "Add a new user. Argument format: `<username>:[admin]:<password>`")
	flag.StringVar(&flagDelUser, "del-user", "", "Remove user by username")
	flag.Parse()

	if flagConfigFile != "" {
		data, err := os.ReadFile(flagConfigFile)
		if err != nil {
			log.Fatal(err)
		}
		if err := json.Unmarshal(data, &programConfig); err != nil {
			log.Fatal(err)
		}
	}

	var err error
	// This might need to change for other databases:
	db, err = sqlx.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", programConfig.DB))
	if err != nil {
		log.Fatal(err)
	}

	// Only for sqlite, not needed for any other database:
	db.SetMaxOpenConns(1)

	// Initialize sub-modules...

	if !programConfig.DisableAuthentication {
		if err := auth.Init(db, programConfig.LdapConfig); err != nil {
			log.Fatal(err)
		}

		if flagNewUser != "" {
			if err := auth.AddUserToDB(db, flagNewUser); err != nil {
				log.Fatal(err)
			}
		}
		if flagDelUser != "" {
			if err := auth.DelUserFromDB(db, flagDelUser); err != nil {
				log.Fatal(err)
			}
		}

		if flagSyncLDAP {
			auth.SyncWithLDAP(db)
		}
	} else if flagNewUser != "" || flagDelUser != "" {
		log.Fatalln("arguments --add-user and --del-user can only be used if authentication is enabled")
	}

	if err := config.Init(db, !programConfig.DisableAuthentication, programConfig.UiDefaults, programConfig.JobArchive); err != nil {
		log.Fatal(err)
	}

	if err := metricdata.Init(programConfig.JobArchive, programConfig.DisableArchive); err != nil {
		log.Fatal(err)
	}

	if flagReinitDB {
		if err := initDB(db, programConfig.JobArchive); err != nil {
			log.Fatal(err)
		}
	}

	if flagStopImmediately {
		return
	}

	// Build routes...

	resolver := &graph.Resolver{DB: db}
	graphQLEndpoint := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	graphQLPlayground := playground.Handler("GraphQL playground", "/query")
	restApi := &api.RestApi{
		DB:             db,
		Resolver:       resolver,
		AsyncArchiving: programConfig.AsyncArchiving,
	}

	handleGetLogin := func(rw http.ResponseWriter, r *http.Request) {
		templates.Render(rw, r, "login", &templates.Page{
			Title: "Login",
			Login: &templates.LoginPage{},
		})
	}

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		templates.Render(rw, r, "404", &templates.Page{
			Title: "Not found",
		})
	})

	r.Handle("/playground", graphQLPlayground)
	r.Handle("/login", auth.Login(db)).Methods(http.MethodPost)
	r.HandleFunc("/login", handleGetLogin).Methods(http.MethodGet)
	r.HandleFunc("/logout", auth.Logout).Methods(http.MethodPost)

	secured := r.PathPrefix("/").Subrouter()
	if !programConfig.DisableAuthentication {
		secured.Use(auth.Auth)
	}
	secured.Handle("/query", graphQLEndpoint)

	secured.HandleFunc("/config.json", config.ServeConfig).Methods(http.MethodGet)

	secured.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		conf, err := config.GetUIConfig(r)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		infos := map[string]interface{}{
			"clusters": config.Clusters,
			"username": "",
			"admin":    true,
		}

		if user := auth.GetUser(r.Context()); user != nil {
			infos["username"] = user.Username
			infos["admin"] = user.IsAdmin
		}

		templates.Render(rw, r, "home", &templates.Page{
			Title:  "ClusterCockpit",
			Config: conf,
			Infos:  infos,
		})
	})

	monitoringRoutes(secured, resolver)
	restApi.MountRoutes(secured)

	r.PathPrefix("/").Handler(http.FileServer(http.Dir(programConfig.StaticFiles)))
	handler := handlers.CORS(
		handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
		handlers.AllowedMethods([]string{"GET", "POST", "HEAD", "OPTIONS"}),
		handlers.AllowedOrigins([]string{"*"}))(handlers.LoggingHandler(os.Stdout, handlers.CompressHandler(r)))

	// Start http or https server
	if programConfig.HttpsCertFile != "" && programConfig.HttpsKeyFile != "" {
		log.Printf("HTTPS server running at %s...", programConfig.Addr)
		err = http.ListenAndServeTLS(programConfig.Addr, programConfig.HttpsCertFile, programConfig.HttpsKeyFile, handler)
	} else {
		log.Printf("HTTP server running at %s...", programConfig.Addr)
		err = http.ListenAndServe(programConfig.Addr, handler)
	}
	log.Fatal(err)
}

func monitoringRoutes(router *mux.Router, resolver *graph.Resolver) {
	router.HandleFunc("/monitoring/jobs/", func(rw http.ResponseWriter, r *http.Request) {
		conf, err := config.GetUIConfig(r)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		filterPresets := map[string]interface{}{}
		query := r.URL.Query()
		if query.Get("tag") != "" {
			filterPresets["tagId"] = query.Get("tag")
		}
		if query.Get("cluster") != "" {
			filterPresets["clusterId"] = query.Get("cluster")
		}
		if query.Get("project") != "" {
			filterPresets["projectId"] = query.Get("project")
		}
		if query.Get("running") == "true" {
			filterPresets["isRunning"] = true
		}
		if query.Get("running") == "false" {
			filterPresets["isRunning"] = false
		}
		if query.Get("from") != "" && query.Get("to") != "" {
			filterPresets["startTime"] = map[string]string{
				"from": query.Get("from"),
				"to":   query.Get("to"),
			}
		}

		templates.Render(rw, r, "monitoring/jobs/", &templates.Page{
			Title:         "Jobs - ClusterCockpit",
			Config:        conf,
			FilterPresets: filterPresets,
		})
	})

	router.HandleFunc("/monitoring/job/{id:[0-9]+}", func(rw http.ResponseWriter, r *http.Request) {
		conf, err := config.GetUIConfig(r)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		id := mux.Vars(r)["id"]
		job, err := resolver.Query().Job(r.Context(), id)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusNotFound)
			return
		}

		templates.Render(rw, r, "monitoring/job/", &templates.Page{
			Title:  fmt.Sprintf("Job %d - ClusterCockpit", job.JobID),
			Config: conf,
			Infos: map[string]interface{}{
				"id":        id,
				"jobId":     job.JobID,
				"clusterId": job.Cluster,
			},
		})
	})

	router.HandleFunc("/monitoring/users/", func(rw http.ResponseWriter, r *http.Request) {
		conf, err := config.GetUIConfig(r)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		templates.Render(rw, r, "monitoring/users/", &templates.Page{
			Title:  "Users - ClusterCockpit",
			Config: conf,
		})
	})

	router.HandleFunc("/monitoring/user/{id}", func(rw http.ResponseWriter, r *http.Request) {
		conf, err := config.GetUIConfig(r)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		id := mux.Vars(r)["id"]
		// TODO: One could check if the user exists, but that would be unhelpfull if authentication
		// is disabled or the user does not exist but has started jobs.

		templates.Render(rw, r, "monitoring/user/", &templates.Page{
			Title:  fmt.Sprintf("User %s - ClusterCockpit", id),
			Config: conf,
			Infos:  map[string]interface{}{"userId": id},
		})
	})

	router.HandleFunc("/monitoring/analysis/", func(rw http.ResponseWriter, r *http.Request) {
		conf, err := config.GetUIConfig(r)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		filterPresets := map[string]interface{}{}
		query := r.URL.Query()
		if query.Get("cluster") != "" {
			filterPresets["clusterId"] = query.Get("cluster")
		}

		templates.Render(rw, r, "monitoring/analysis/", &templates.Page{
			Title:         "Analysis View - ClusterCockpit",
			Config:        conf,
			FilterPresets: filterPresets,
		})
	})

	router.HandleFunc("/monitoring/systems/", func(rw http.ResponseWriter, r *http.Request) {
		conf, err := config.GetUIConfig(r)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		filterPresets := map[string]interface{}{}
		query := r.URL.Query()
		if query.Get("cluster") != "" {
			filterPresets["clusterId"] = query.Get("cluster")
		}

		templates.Render(rw, r, "monitoring/systems/", &templates.Page{
			Title:         "System View - ClusterCockpit",
			Config:        conf,
			FilterPresets: filterPresets,
		})
	})

	router.HandleFunc("/monitoring/node/{clusterId}/{nodeId}", func(rw http.ResponseWriter, r *http.Request) {
		conf, err := config.GetUIConfig(r)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		vars := mux.Vars(r)
		templates.Render(rw, r, "monitoring/node/", &templates.Page{
			Title:  fmt.Sprintf("Node %s - ClusterCockpit", vars["nodeId"]),
			Config: conf,
			Infos: map[string]interface{}{
				"nodeId":    vars["nodeId"],
				"clusterId": vars["clusterId"],
			},
		})
	})
}
