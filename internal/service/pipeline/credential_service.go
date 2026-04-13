package pipeline

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
)

// 加密密钥（生产环境应从配置读取）
var encryptionKey = []byte("devops-pipeline-credential-key32")

// CredentialService 凭证服务
type CredentialService struct {
	db *gorm.DB
}

// NewCredentialService 创建凭证服务
func NewCredentialService(db *gorm.DB) *CredentialService {
	return &CredentialService{db: db}
}

// List 获取凭证列表
func (s *CredentialService) List(ctx context.Context) ([]dto.CredentialItem, error) {
	var credentials []models.PipelineCredential

	if err := s.db.Order("name").Find(&credentials).Error; err != nil {
		return nil, err
	}

	items := make([]dto.CredentialItem, 0, len(credentials))
	for _, c := range credentials {
		items = append(items, dto.CredentialItem{
			ID:          c.ID,
			Name:        c.Name,
			Type:        c.Type,
			Description: c.Description,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		})
	}

	return items, nil
}

// Create 创建凭证
func (s *CredentialService) Create(ctx context.Context, req *dto.CredentialRequest) error {
	// 加密数据
	encrypted, err := encrypt(req.Data)
	if err != nil {
		return err
	}

	credential := &models.PipelineCredential{
		Name:          req.Name,
		Type:          req.Type,
		Description:   req.Description,
		DataEncrypted: encrypted,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return s.db.Create(credential).Error
}

// Update 更新凭证
func (s *CredentialService) Update(ctx context.Context, req *dto.CredentialRequest) error {
	var credential models.PipelineCredential
	if err := s.db.First(&credential, req.ID).Error; err != nil {
		return err
	}

	credential.Name = req.Name
	credential.Type = req.Type
	credential.Description = req.Description
	credential.UpdatedAt = time.Now()

	if req.Data != "" {
		encrypted, err := encrypt(req.Data)
		if err != nil {
			return err
		}
		credential.DataEncrypted = encrypted
	}

	return s.db.Save(&credential).Error
}

// Delete 删除凭证
func (s *CredentialService) Delete(ctx context.Context, id uint) error {
	return s.db.Delete(&models.PipelineCredential{}, id).Error
}

// GetDecrypted 获取解密后的凭证（内部使用）
func (s *CredentialService) GetDecrypted(ctx context.Context, id uint) (string, error) {
	var credential models.PipelineCredential
	if err := s.db.First(&credential, id).Error; err != nil {
		return "", err
	}

	return decrypt(credential.DataEncrypted)
}

// CredentialData 凭证数据
type CredentialData struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
	SSHKey   string `json:"ssh_key"`
}

// GetDecryptedData 获取解密后的凭证数据（内部使用）
func (s *CredentialService) GetDecryptedData(ctx context.Context, id uint) (*CredentialData, error) {
	var credential models.PipelineCredential
	if err := s.db.First(&credential, id).Error; err != nil {
		return nil, err
	}

	decrypted, err := decrypt(credential.DataEncrypted)
	if err != nil {
		return nil, err
	}

	// 解析 JSON 格式的凭证数据
	var data CredentialData
	if err := json.Unmarshal([]byte(decrypted), &data); err != nil {
		// 如果不是 JSON 格式，尝试作为简单密码处理
		data.Password = decrypted
	}

	return &data, nil
}

// encrypt 加密
func encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 解密
func decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", err
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
