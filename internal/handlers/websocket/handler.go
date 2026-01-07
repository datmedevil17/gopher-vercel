package websocket

import (
	"log"
	"net/http"

	"deployment-platform/internal/services/websocket"

	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

type Handler struct {
	hub *websocket.Hub
}

func NewHandler(hub *websocket.Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) HandleLogs(c *gin.Context) {
	deploymentID := c.Param("id")
	if deploymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Deployment ID required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	h.hub.RegisterClient(conn, deploymentID)

	// Keep connection open until client disconnects
	// The Hub handles writing messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
