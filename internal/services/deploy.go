package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"deployment-platform/internal/models"

	"deployment-platform/internal/services/websocket"

	"github.com/go-git/go-git/v5"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
)

type DeployService struct {
	db        *gorm.DB
	rmq       *RabbitMQ
	s3Service *S3Service
	hub       *websocket.Hub
}

func NewDeployService(db *gorm.DB, rmq *RabbitMQ, s3 *S3Service, hub *websocket.Hub) *DeployService {
	service := &DeployService{
		db:        db,
		rmq:       rmq,
		s3Service: s3,
		hub:       hub,
	}

	// Start consuming messages
	go service.StartWorker()

	return service
}

func (s *DeployService) QueueDeployment(deployment *models.Deployment) error {
	body, err := json.Marshal(map[string]interface{}{
		"deploy_id": deployment.DeployID,
		"repo_url":  deployment.RepoURL,
	})
	if err != nil {
		return err
	}

	return s.rmq.Publish("deployments", body)
}

func (s *DeployService) StartWorker() {
	msgs, err := s.rmq.Consume("deployments")
	if err != nil {
		log.Fatal("Failed to start worker:", err)
	}

	log.Println("Worker started, waiting for messages...")

	for msg := range msgs {
		s.processDeployment(msg)
	}
}

func (s *DeployService) processDeployment(msg amqp.Delivery) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Body, &data); err != nil {
		log.Println("Error unmarshaling message:", err)
		msg.Nack(false, false)
		return
	}

	deployID := data["deploy_id"].(string)
	repoURL := data["repo_url"].(string)

	log.Printf("Processing deployment: %s", deployID)

	var deployment models.Deployment
	if err := s.db.Where("deploy_id = ?", deployID).First(&deployment).Error; err != nil {
		log.Println("Error finding deployment:", err)
		msg.Nack(false, false)
		return
	}

	// Clone repository
	deployment.Status = "cloning"
	s.db.Save(&deployment)

	tmpDir := filepath.Join("/tmp", deployID)
	if err := s.cloneRepo(repoURL, tmpDir); err != nil {
		deployment.Status = "failed"
		deployment.ErrorMsg = fmt.Sprintf("Clone failed: %v", err)
		s.db.Save(&deployment)
		msg.Ack(false)
		return
	}

	// Upload files to S3
	deployment.Status = "uploading"
	s.db.Save(&deployment)

	if err := s.s3Service.UploadDirectory(tmpDir, fmt.Sprintf("source/%s", deployID)); err != nil {
		deployment.Status = "failed"
		deployment.ErrorMsg = fmt.Sprintf("Upload failed: %v", err)
		s.db.Save(&deployment)
		msg.Ack(false)
		return
	}

	// Build project
	deployment.Status = "building"
	s.db.Save(&deployment)
	s.hub.BroadcastLog(deployID, "Starting build process...")

	buildLog, err := s.buildProject(tmpDir, deployID)
	deployment.BuildLog = buildLog
	if err != nil {
		deployment.Status = "failed"
		deployment.ErrorMsg = fmt.Sprintf("Build failed: %v", err)
		s.db.Save(&deployment)
		msg.Ack(false)
		return
	}

	// Upload dist files
	distDir := filepath.Join(tmpDir, "dist")
	if err := s.s3Service.UploadDirectory(distDir, fmt.Sprintf("dist/%s", deployID)); err != nil {
		deployment.Status = "failed"
		deployment.ErrorMsg = fmt.Sprintf("Dist upload failed: %v", err)
		s.db.Save(&deployment)
		msg.Ack(false)
		return
	}

	// Clean up
	os.RemoveAll(tmpDir)

	// Mark as deployed
	deployment.Status = "deployed"
	s.db.Save(&deployment)

	msg.Ack(false)
	log.Printf("Deployment completed: %s", deployID)
}

func (s *DeployService) cloneRepo(repoURL, destPath string) error {
	_, err := git.PlainClone(destPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	return err
}

func (s *DeployService) buildProject(projectPath, deployID string) (string, error) {
	// Install dependencies
	s.hub.BroadcastLog(deployID, "Installing dependencies...")
	installCmd := exec.Command("npm", "install")
	installCmd.Dir = projectPath

	installOutput, err := s.runCommandWithStreaming(installCmd, deployID)
	if err != nil {
		return installOutput, err
	}

	// Build project
	s.hub.BroadcastLog(deployID, "Building project...")
	buildCmd := exec.Command("npm", "run", "build")
	buildCmd.Dir = projectPath

	buildOutput, err := s.runCommandWithStreaming(buildCmd, deployID)

	fullLog := installOutput + "\n" + buildOutput
	return fullLog, err
}

func (s *DeployService) runCommandWithStreaming(cmd *exec.Cmd, deployID string) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var output []byte
	buf := make([]byte, 1024)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			output = append(output, chunk...)
			s.hub.BroadcastLog(deployID, string(chunk))
		}
		if err != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		return string(output), err
	}

	return string(output), nil
}
