package main

import (
	"bugmaschine/e6-cache/logging"
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"bugmaschine/e6-cache/signer"

	"github.com/getkin/kin-openapi/openapi3"

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

	//go:embed openapi.yaml
	openApiRoutes []byte // embedded OpenAPI routes, used to dynamically register the routes in the gin router.
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

	// load e621 routes
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(openApiRoutes)
	if err != nil {
		log.Fatalf("Failed to load OpenAPI: %v (make sure to run 'go embed')", err)
	}

	// create a regex to convert OpenAPI path parameters {id} to gins format :id
	re := regexp.MustCompile(`\{(.+?)\}`)

	for _, path := range doc.Paths.InMatchingOrder() {
		pathItem := doc.Paths.Find(path)
		// convert OpenAPI parameter syntax to Gin parameter syntax, with the problem being that gin has problems supporting that
		convertedPath := re.ReplaceAllString(path, ":$1")
		for method := range pathItem.Operations() {
			router.Handle(method, convertedPath, proxyAndTransform)
		}
	}

	logging.Info(fmt.Sprintf("Registered %d routes from OpenAPI spec", len(doc.Paths.InMatchingOrder())))

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
