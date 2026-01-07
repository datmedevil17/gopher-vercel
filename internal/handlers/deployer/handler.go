package deployer

import (
	"net/http"

	"deployment-platform/internal/services/deployer"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service deployer.Service
}

func NewHandler(service deployer.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Deploy(c *gin.Context) {
	var req DeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (if authenticated)
	userID := uint(0)
	if val, exists := c.Get("user_id"); exists {
		userID = val.(uint)
	}

	deployment, err := h.service.CreateDeployment(c.Request.Context(), userID, req.RepoURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           deployment.DeployID,
		"status":       deployment.Status,
		"deployed_url": deployment.DeployedURL,
	})
}

func (h *Handler) GetStatus(c *gin.Context) {
	deployID := c.Param("id")

	deployment, err := h.service.GetDeploymentStatus(c.Request.Context(), deployID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

func (h *Handler) GetDeployments(c *gin.Context) {
	userID := c.GetUint("user_id")

	deployments, err := h.service.GetUserDeployments(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployments)
}

func (h *Handler) DeleteDeployment(c *gin.Context) {
	deployID := c.Param("id")
	userID := c.GetUint("user_id")

	err := h.service.DeleteDeployment(c.Request.Context(), deployID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deployment deleted successfully"})
}
