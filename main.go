package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"server-sent-events/event"
	"server-sent-events/tutorial"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
)

func main() {
	// httpSSE()
	// fiberSSE()
	// ginSSEAndFS()
	// ginSSE()
	ginSSETutorial()
}

// SSE with gin tutorial
// https://github.com/gin-gonic/examples/blob/master/server-sent-event/main.go
func HeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Next()
	}
}

func ginSSETutorial() {
	router := gin.Default()

	// Initialize new streaming server
	stream := tutorial.NewServer()

	// We are streaming current time to clients in the interval 10 seconds
	go func() {
		for {
			time.Sleep(time.Second * 10)
			now := time.Now().Format("2006-01-02 15:04:05")
			currentTime := fmt.Sprintf("The Current Time Is %v", now)

			// Send current time to clients message channel
			stream.Message <- currentTime
		}
	}()

	// Basic Authentication
	authorized := router.Group("/", gin.BasicAuth(gin.Accounts{
		"admin": "admin123", // username : admin, password : admin123
	}))

	// Authorized client can stream the event
	// Add event-streaming headers
	authorized.GET("/stream", HeadersMiddleware(), stream.ServeHTTP(), func(c *gin.Context) {
		v, ok := c.Get("clientChan")
		if !ok {
			return
		}
		clientChan, ok := v.(tutorial.ClientChan)
		if !ok {
			return
		}
		c.Stream(func(w io.Writer) bool {
			// Stream message to client from message channel
			if msg, ok := <-clientChan; ok {
				c.SSEvent("message", msg)
				return true
			}
			return false
		})
	})

	// Parse Static files
	router.StaticFile("/", "./public/index.html")

	router.Run(":8085")
}

// SSE with gin
// https://pascalallen.medium.com/streaming-server-sent-events-with-go-8cc1f615d561
func ginSSE() {
	ch := make(chan string)

	router := gin.Default()
	router.POST("/event-stream", func(c *gin.Context) {
		event.HandleEventStreamPost(c, ch)
	})

	router.GET("/event-stream", func(c *gin.Context) {
		event.HandleEventStreamGet(c, ch)
	})

	log.Fatalf("error running HTTP server: %s\n", router.Run(":9990"))
}

// SSE with gin using embed.FS
var f embed.FS

func ginSSEAndFS() {
	router := gin.Default()
	templ := template.Must(template.New("").ParseFS(f, "templates/*.tpl"))
	router.SetHTMLTemplate(templ)
	router.StaticFS("/public", http.FS(f))

	// SSE endpoint
	router.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.tpl", nil)
	})
	router.GET("/progress", progressor)

	// Start the server
	if err := router.Run(":8080"); err != nil {
		panic(err)
	}
}

func progressor(ctx *gin.Context) {
	noOfExecution := 10
	progress := 0
	for progress <= noOfExecution {
		progressPercentage := float64(progress) / float64(noOfExecution) * 100

		ctx.SSEvent("progress", map[string]interface{}{
			"currentTask":        progress,
			"progressPercentage": progressPercentage,
			"noOftasks":          noOfExecution,
			"completed":          false,
		})
		// Flush the response to ensure the data is sent immediately
		ctx.Writer.Flush()

		progress += 1
		time.Sleep(2 * time.Second)
	}

	ctx.SSEvent("progress", map[string]interface{}{
		"completed":          true,
		"progressPercentage": 100,
	})

	// Flush the response to ensure the data is sent immediately
	ctx.Writer.Flush()
}

// SSE with fiber
func fiberSSE() {
	app := fiber.New()
	app.Get("/events", adaptor.HTTPHandler(handler(dashboardHandler)))
	app.Listen(":8080")
}

type Client struct {
	name   string
	events chan *DashBoard
}

type DashBoard struct {
	User uint
}

func handler(f http.HandlerFunc) http.Handler {
	return http.HandlerFunc(f)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	client := &Client{name: r.RemoteAddr, events: make(chan *DashBoard, 10)}
	go updateDashboard(client)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	timeout := time.After(1 * time.Second)
	select {
	case ev := <-client.events:
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.Encode(ev)
		fmt.Fprintf(w, "data: %v\n\n", buf.String())
		fmt.Printf("data: %v\n", buf.String())
	case <-timeout:
		fmt.Fprintf(w, ": nothing to sent\n\n")
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func updateDashboard(client *Client) {
	for {
		db := &DashBoard{
			User: uint(rand.Uint32()),
		}
		client.events <- db
	}
}

// SSE with http
func httpSSE() {
	http.HandleFunc("/events", eventsHandler)
	http.ListenAndServe(":8080", nil)
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers to allow all origins. You may want to restrict this to specific origins in a production environment.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Simulate sending events (you can replace this with real data)
	for i := 0; i < 10; i++ {
		fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf("Event %d", i))
		time.Sleep(2 * time.Second)
		w.(http.Flusher).Flush()
	}

	// Simulate closing the connection
	closeNotify := w.(http.CloseNotifier).CloseNotify()
	<-closeNotify
}
