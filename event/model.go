package event

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type EventStreamRequest struct {
	Message string `form:"message" json:"message" binding:"required,max=100"`
}

type JSendFailResponse[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data"`
}

type JSendSuccessResponse[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data,omitempty"`
}

func HandleEventStreamPost(c *gin.Context, ch chan string) {
	var request EventStreamRequest
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, JSendFailResponse[string]{
			Status: "fail",
			Data:   err.Error(),
		})
		return
	}

	ch <- request.Message

	c.JSON(http.StatusCreated, JSendSuccessResponse[interface{}]{
		Status: "success",
		Data:   request.Message,
	})
}

func HandleEventStreamGet(c *gin.Context, ch chan string) {
	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-ch; ok {
			c.SSEvent("message", msg)
			return true
		}
		return false
	})
}
