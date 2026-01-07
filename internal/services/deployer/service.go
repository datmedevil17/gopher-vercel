package deployer

import (
	"context"
	"fmt"

	"deployment-platform/internal/models"
	"deployment-platform/internal/services"
	"deployment-platform/internal/utils"

	"gorm.io/gorm"
)

type Service interface {
	CreateDeployment(ctx context.Context, userID uint, repoURL string) (*models.Deployment, error)
	GetDeploymentStatus(ctx context.Context, deployID string) (*models.Deployment, error)
	GetUserDeployments(ctx context.Context, userID uint) ([]models.Deployment, error)
	DeleteDeployment(ctx context.Context, deployID string, userID uint) error
}

type service struct {
	db            *gorm.DB
	deployService *services.DeployService
	baseDomain    string
}

func NewService(db *gorm.DB, deployService *services.DeployService, baseDomain string) Service {
	return &service{
		db:            db,
		deployService: deployService,
		baseDomain:    baseDomain,
	}
}

func (s *service) CreateDeployment(ctx context.Context, userID uint, repoURL string) (*models.Deployment, error) {
	deployID := utils.GenerateID(8)

	deployment := &models.Deployment{
		UserID:      userID,
		DeployID:    deployID,
		RepoURL:     repoURL,
		Status:      "pending",
		DeployedURL: fmt.Sprintf("http://%s.%s", deployID, s.baseDomain),
	}

	if err := s.db.Create(deployment).Error; err != nil {
		return nil, err
	}

	// Send to RabbitMQ for processing
	if err := s.deployService.QueueDeployment(deployment); err != nil {
		deployment.Status = "failed"
		deployment.ErrorMsg = err.Error()
		s.db.Save(deployment)
		return nil, err
	}

	return deployment, nil
}

func (s *service) GetDeploymentStatus(ctx context.Context, deployID string) (*models.Deployment, error) {
	var deployment models.Deployment
	if err := s.db.Where("deploy_id = ?", deployID).First(&deployment).Error; err != nil {
		return nil, err
	}
	return &deployment, nil
}

func (s *service) GetUserDeployments(ctx context.Context, userID uint) ([]models.Deployment, error) {
	var deployments []models.Deployment
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&deployments).Error; err != nil {
		return nil, err
	}
	return deployments, nil
}

func (s *service) DeleteDeployment(ctx context.Context, deployID string, userID uint) error {
	result := s.db.Where("deploy_id = ? AND user_id = ?", deployID, userID).Delete(&models.Deployment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("deployment not found or unauthorized")
	}
	return nil
}
