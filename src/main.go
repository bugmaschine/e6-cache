package main

import (
	"bugmaschine/e6-cache/logging"
	"context"
	_ "embed"
	"os"
	"strconv"
	"time"

	"bugmaschine/e6-cache/signer"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func isDebug() bool {
	return debugMode == "true"
}

var (
	debugMode     string = "false"
	Database      DB
	useragentBase = "e6-cache (https://github.com/bugmaschine/e6-cache)"
	port          = ":8080"
	Key           []byte            // gets randomly generated every launch, and used for signing the urls.
	maxCacheAge   = 1 * time.Hour   // idk what's a good value, but 1 hours seems enough
	Signer        *signer.Signer    // feel free to sugest a better name
	globalTimeout = 5 * time.Second // global timeout for requests to e6, if it takes longer than this, we assume the request failed.

	// env stuff

	// S3
	S3_BUCKET_NAME string
	S3_REGION      string
	S3_ACCESS_KEY  string
	S3_SECRET_KEY  string
	S3_ENDPOINT    string
	S3             S3Service

	// PostgreSQL
	DB_HOST string
	DB_PORT int
	DB_NAME string
	DB_USER string
	DB_PASS string

	// Proxy settings
	PROXY_URL  string
	baseURL    string
	PROXY_AUTH string
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		logging.Warn("Error loading .env file")
	}

	S3_BUCKET_NAME = os.Getenv("S3_BUCKET")
	S3_REGION = os.Getenv("S3_REGION")
	S3_ACCESS_KEY = os.Getenv("S3_ACCESS_KEY")
	S3_SECRET_KEY = os.Getenv("S3_SECRET_KEY")
	S3_ENDPOINT = os.Getenv("S3_ENDPOINT")

	DB_HOST = os.Getenv("DB_HOST")
	i, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		logging.Fatal("Error converting DB_PORT to int")
	}
	DB_PORT = i
	DB_NAME = os.Getenv("DB_NAME")
	DB_USER = os.Getenv("DB_USER")
	DB_PASS = os.Getenv("DB_PASS")

	// Proxy Settings
	PROXY_URL = os.Getenv("PROXY_URL")
	baseURL = os.Getenv("E6_BASE")
	PROXY_AUTH = os.Getenv("PROXY_AUTH")

	if PROXY_AUTH != "" {
		logging.Debug("Proxy auth is enabled with key: ", PROXY_AUTH)
	} else {
		logging.Debug("Proxy auth is disabled")
	}

}

func main() {
	logging.Setup(".", isDebug())

	logging.Info("Starting e6-cache...")
	loadEnv()

	// generate signing key
	Key = signer.GenerateSecretKey()
	Signer = signer.NewSigner(Key)
	logging.Debug("Generated key: ", Key)

	// setup db
	logging.Info("Connecting to DB...")
	d, err := newDB(DB_HOST, DB_NAME, DB_USER, DB_PASS, DB_PORT)
	if err != nil {
		logging.Info("Failed to connect to DB (is it up?): ", err)
		return
	}
	Database = d
	logging.Info("Connected to DB!")

	// setup s3
	logging.Info("Connecting to S3...")
	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()
	s3Svc, err := NewS3Service(ctx, S3_REGION, S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET_NAME)
	if err != nil {
		logging.Fatal("Failed to connect to S3: ", err)
	}
	S3 = *s3Svc
	logging.Info("Connected to S3!")

	// create gin router
	router := gin.Default()

	if !isDebug() {
		gin.SetMode(gin.ReleaseMode)
	}

	router.ForwardedByClientIP = true
	router.Use(gin.Recovery())

	// Routes implementing caching

	router.GET("/posts.json", proxyAndTransform)
	router.GET("/posts/:Post_ID.json", proxyAndTransform)
	router.GET("/pools.json", proxyAndTransform)
	router.GET("/pools/:Pool_ID", proxyAndTransform)
	router.GET("/comments.json", proxyAndTransform) // this for reason just returns posts?????????

	// Routes that could be maybe saved

	router.GET("/notes.json", proxyAndTransform)
	router.GET("/wiki_pages.json", proxyAndTransform)
	router.GET("/post_flags.json", proxyAndTransform)

	// Routes just needing proxying and have make no sense being cached

	router.POST("/favorites.json", proxyAndTransform)
	router.POST("/uploads.json", proxyAndTransform)
	router.POST("/post_flags.json", proxyAndTransform)
	router.DELETE("/favorites/:Post_ID", proxyAndTransform)
	router.POST("/notes.json", proxyAndTransform)
	router.PUT("/notes/*path", proxyAndTransform)
	router.DELETE("/notes/:Note_ID", proxyAndTransform)
	router.POST("/posts/:Post_ID/votes.json", proxyAndTransform)
	router.GET("/forum_topics.json", proxyAndTransform)
	router.GET("/forum_posts.json", proxyAndTransform)
	router.POST("/comments.json", proxyAndTransform)
	router.PUT("/comments/:Comment_ID", proxyAndTransform)
	router.DELETE("/comments/:Comment_ID", proxyAndTransform)
	router.POST("/comments/:Comment_ID/votes.json", proxyAndTransform)
	router.PUT("/users/:User_ID.json", proxyAndTransform)
	router.GET("/users/:User_ID.json", proxyAndTransform)
	router.GET("/tags/*path", proxyAndTransform)

	// potentially just forward to the baseUrl
	router.PATCH("/posts/:Post_ID", proxyAndTransform)

	// Proxy files from S3, if not save them.
	router.GET("/proxy/:File_ID", proxyFile)

	router.GET("/", func(c *gin.Context) {
		c.String(200, "e6-cache is running. Use this as the instance in your preffered client.\n"+
			"Make sure to set the base URL in your client to: "+PROXY_URL+"\n"+
			"Server is caching following url: "+baseURL)
	})

	logging.Info("Started router at ", port)
	router.Run(port)

}
