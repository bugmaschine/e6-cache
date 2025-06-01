package main

import (
	"bugmaschine/e6-cache/dualreader"
	"bugmaschine/e6-cache/logging"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

var (
	// honestly just copying only the auth header and maybe adding compression headers might be far easier.
	// TODO: change this someday
	headersToSkip = []string{
		"user-agent", "via", "host", "content-Length", "x-forwarded-for", "x-real-ip", "x-forwarded-host", "x-forwarded-proto", "x-forwarded-for",
	}
)

func makeProxyLink(original string) string {
	if original == "" { // in case the input is empty, just return an empty string
		logging.Warn("Received empty URL for proxying")
		return ""
	}
	// basically get the part after "data/" in the url
	re := regexp.MustCompile(`data/(.+)`)
	match := re.FindStringSubmatch(original)

	sig := Signer.Sign(original)

	// We sign the url so a malicious attacker can't just change the url to download any file. But this also means that the signature changes every restart.
	encodedUrl := base64.URLEncoding.EncodeToString([]byte(original))

	// if the route changes, we need to update this
	proxiedURL := PROXY_URL + "/proxy/" + encodedUrl + "?sig=" + sig
	logging.Info("Creating proxy url for file: %v | ID: %v | Proxied URL: %v", original, match[1], proxiedURL)

	return proxiedURL
}

func proxyAndTransform(c *gin.Context) {

	logging.Debug("Headers: %v", c.Request.Header)

	auth := c.Request.Header.Get("Authorization")

	requestUsername := ""
	// if proxy auth is enabled, check for auth header
	if PROXY_AUTH != "" {
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// an auth header should normaly look like this: base64 hashed username:password
		// ours should look like this username:proxy_auth:password
		// the idea is that the user enters their username and then a colon and then the proxy auth. Because most clients dont support colons in the api key, but will accept it in user names
		auth = strings.TrimPrefix(auth, "Basic ")
		decodedAuth, err := base64.StdEncoding.DecodeString(auth)
		if err != nil {
			logging.Error("Error decoding auth header: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		// parse it into username, password and proxy auth
		authParts := strings.Split(string(decodedAuth), ":")
		suppliedProxyAuth := authParts[1] // to change the password position you need to change this here. 1 is after the username, 2 is after the proxy auth
		logging.Debug("Parsed Proxy Authorization header: %v", authParts)
		if len(authParts) != 3 || suppliedProxyAuth != PROXY_AUTH {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		c.Request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(authParts[0]+":"+authParts[2])))
	} else {
		// this gets run if the proxy auth function not enabled
		if auth == "" {
			logging.Debug("No Authorization header found, using anonymous user")
			requestUsername = "anonymous"
		}

		// parse the Authorization header
		auth = strings.ReplaceAll(auth, "Basic ", "")
		decodedAuth, _ := base64.URLEncoding.DecodeString(auth)
		splitAuth := strings.Split(string(decodedAuth), ":")
		requestUsername = splitAuth[0] // 0 is username, 1 is api key
	}

	// Construct full target URL
	originalURL := baseURL + c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		originalURL += "?" + c.Request.URL.RawQuery
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	c.Request.Body.Close()

	// Create the proxied request with the copied body
	req, err := http.NewRequest(c.Request.Method, originalURL, bytes.NewReader(bodyBytes))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create proxy request"})
		return
	}

	// Copy headers
	copyHeaders(c.Request.Header, req.Header)

	// useragent stuff
	setUseragent(requestUsername, req)

	logging.Debug("Host: %v", c.Request.Host)
	logging.Debug("Proxied Headers: %v", req.Header)

	// Perform request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "Failed to reach backend"})
		return
	}
	defer resp.Body.Close()

	var reader io.ReadCloser

	// https://stackoverflow.com/questions/13130341/reading-gzipped-http-response-in-go
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		logging.Debug("Server sent gzip compressed response")

		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "gzip decompression failed"})
			return
		}
		defer reader.Close()

	case "deflate":
		logging.Debug("Server sent deflate compressed response")

		reader = flate.NewReader(resp.Body)
		defer reader.Close()

	case "compress":
		logging.Debug("Server sent standard compressed response")

		reader, err = zlib.NewReader(resp.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "compress decompression failed"})
			return
		}
		defer reader.Close()

	case "br":
		logging.Debug("Server sent brotli compressed response")

		reader = io.NopCloser(brotli.NewReader(resp.Body))
		defer reader.Close()

	default:
		logging.Debug("Server sent uncompressed response")
		reader = resp.Body
	}

	// Read response body
	respBody, err := io.ReadAll(reader)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body"})
		return
	}

	// check for rate limit
	if resp.StatusCode == 501 {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
		return
	}

	logging.Debug("Response Body: %v", string(respBody))

	switch {
	case strings.HasSuffix(c.Request.URL.Path, "/comments.json") && c.Query("search[post_id]") != "": // specific post comments are returned differently
		var comments []Comment

		if err := json.Unmarshal(respBody, &comments); err != nil {
			// it failed to unmahrshal because the api returned "commments": []
			var comments CommentsResponse
			if err := json.Unmarshal(respBody, &comments); err != nil {

				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid response format", "ok": false})
				return
			}
			respBody, _ = json.Marshal(comments)
			return
		}

		logging.Info("Saving %v comments", len(comments))
		Database.SaveComments(comments)
		respBody, _ = json.Marshal(comments)
	case strings.HasSuffix(c.Request.URL.Path, "/posts.json") || strings.HasSuffix(c.Request.URL.Path, "/comments.json"): // comments and posts seem to be the same thing
		var posts PostsResponse

		if err := json.Unmarshal(respBody, &posts); err != nil {
			logging.Debug("Response Body: %v", string(respBody))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid response format", "ok": false})
			return
		}

		for i := range posts.Posts {
			ProcessPost(c, &posts.Posts[i])
		}

		respBody, _ = json.Marshal(posts)
	case strings.HasPrefix(c.Request.URL.Path, "/posts/"):
		var post PostResponse

		if err := json.Unmarshal(respBody, &post); err != nil {

			logging.Debug("Response Body: %v", string(respBody))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid response format", "ok": false})
			return
		}

		ProcessPost(c, &post.Post)

		respBody, _ = json.Marshal(post)
	case strings.HasSuffix(c.Request.URL.Path, "/pools.json"):
		var pools []Pool

		if err := json.Unmarshal(respBody, &pools); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid response format", "ok": false})
			return
		}

		for _, pool := range pools {
			// Store in DB
			ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
			defer cancel()
			Database.UpdatePool(ctx, &pool)
		}

		respBody, _ = json.Marshal(pools)
	case strings.Contains(c.Request.URL.Path, "/pools/"):
		var pool Pool

		if err := json.Unmarshal(respBody, &pool); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid response format", "ok": false})
			return
		}

		// Store in DB
		ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
		defer cancel()
		Database.UpdatePool(ctx, &pool)

		respBody, _ = json.Marshal(pool)
	}

	// Send client response
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

func proxyFile(c *gin.Context) {
	fileID := c.Param("fileId")
	sig := c.Query("sig")

	if sig == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing signature", "ok": false})
		return
	}

	url, err := base64.URLEncoding.DecodeString(fileID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID", "ok": false})
		return
	}

	if !Signer.Verify(string(url), sig) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid signature", "ok": false})
		return
	}

	// get the filename from the fileID
	re := regexp.MustCompile(`/data/(.*)`)
	matches := re.FindStringSubmatch(string(url))

	CleanFileID := matches[1]

	fileExists, err := S3.DoesFileExistInS3(c, string(CleanFileID))
	c.Header("Cache-Control", "public, max-age="+strconv.Itoa(int(maxCacheAge.Seconds())))
	c.Header("Expires", time.Now().Add(time.Duration(maxCacheAge)*time.Second).Format(http.TimeFormat))

	if fileExists && err == nil {
		logging.Info("File exists in S3, downloading: %v", string(url))

		body, err := S3.StreamFromS3(c, string(CleanFileID))
		if err != nil {
			logging.Error("Error downloading from S3: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to download from S3", "ok": false})
			return
		}

		// dont know if streaming from s3 is actually possible, but let's pretend it is
		contentLength, _ := S3.GetContentLength(c, string(CleanFileID))
		if body == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to download from S3", "ok": false})
			return
		}

		var contentType string
		switch ext := filepath.Ext(string(CleanFileID)); ext {
		// Images
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		case ".webp":
			contentType = "image/webp"
		case ".bmp":
			contentType = "image/bmp"
		case ".tiff", ".tif":
			contentType = "image/tiff"

		// Videos
		case ".webm":
			contentType = "video/webm"
		case ".mp4":
			contentType = "video/mp4"
		case ".mov":
			contentType = "video/quicktime"
		case ".avi":
			contentType = "video/x-msvideo"
		case ".mkv":
			contentType = "video/x-matroska"
		case ".flv":
			contentType = "video/x-flv"
		case ".ogv":
			contentType = "video/ogg"

		default:
			contentType = "application/octet-stream"
		}

		c.DataFromReader(200, contentLength, contentType, body, nil)
		return
	}

	// below only gets called when file does not exist in S3
	logging.Debug("File not found in S3. Requesting it.")

	// download the image from the api
	req, _ := http.NewRequest("GET", string(url), nil)

	// i dont think the username is required for downloading files
	setUseragent("", req)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to proxy request", "ok": false})
		return
	}
	defer resp.Body.Close()

	// some magic to handle streaming the response body to S3 and to the user at the same time
	dual := dualreader.NewDualReader(resp.Body)
	r1, r2 := dual.Readers()

	// upload to S3 in the background, while the user is downloading the file
	go func() {
		logging.Info("Uploading to S3: %v", string(CleanFileID))
		err := S3.UploadToS3(c, r1, string(CleanFileID))
		if err != nil {
			logging.Error("Failed to upload to S3: %v", err)
		}
		logging.Info("Upload to S3 complete: %v", string(CleanFileID))
	}()

	// Stream live to user (I hope it's actually streaming)
	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), r2, nil)
}

func copyHeaders(src http.Header, dst http.Header) {
	skip := make(map[string]struct{}, len(headersToSkip))

	for _, h := range headersToSkip {
		// convert to lowercase
		skip[strings.ToLower(h)] = struct{}{}
	}

	for k, vv := range src {
		// skip headers that are in the skip list
		if _, found := skip[strings.ToLower(k)]; found {
			continue // skip this header
		}
		// if not found, add the header to the destination
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func ProcessPost(c *gin.Context, post *Post) {
	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()
	Database.CheckAndInsertPost(ctx, post)

	// Rewrite url to go through the proxy
	post.File.URL = makeProxyLink(post.File.URL)
	post.Preview.URL = makeProxyLink(post.Preview.URL)
	post.Sample.URL = makeProxyLink(post.Sample.URL)
}

func setUseragent(username string, req *http.Request) {
	var useragent string
	if len(username) < 1 { // if it's too small, then forget about it
		useragent = useragentBase
		req.Header.Set("User-Agent", useragent)
		return
	}

	useragent = fmt.Sprintf("%s (Request made on behalf of %s)", useragentBase, username)

	req.Header.Set("User-Agent", useragent)
}
