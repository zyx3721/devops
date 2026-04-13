package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"

	"golang.org/x/crypto/pbkdf2"
	"gorm.io/gorm"

	"devops/internal/models"
)

var (
	ErrEncryptionFailed  = errors.New("加密失败")
	ErrDecryptionFailed  = errors.New("解密失败")
	ErrKeyNotFound       = errors.New("密钥不存在")
	ErrInvalidCiphertext = errors.New("无效的密文")
)

// EncryptionService 加密服务接口
type EncryptionService interface {
	// 数据加密解密
	Encrypt(ctx context.Context, plaintext []byte) (string, error)
	Decrypt(ctx context.Context, ciphertext string) ([]byte, error)

	// 字符串加密解密（便捷方法）
	EncryptString(ctx context.Context, plaintext string) (string, error)
	DecryptString(ctx context.Context, ciphertext string) (string, error)

	// 密钥管理
	GetOrCreateKey(ctx context.Context) (*models.EncryptionKey, error)
	RotateKey(ctx context.Context) error
}

// encryptionServiceImpl 加密服务实现
type encryptionServiceImpl struct {
	db        *gorm.DB
	masterKey []byte // 主密钥，用于加密数据密钥
	keyCache  sync.Map
}

// NewEncryptionService 创建加密服务
func NewEncryptionService(db *gorm.DB, masterKey string) EncryptionService {
	// 使用 PBKDF2 派生密钥，而非简单的零字节填充
	// 这样即使主密钥较短，也能生成安全的 32 字节密钥
	key := deriveKey(masterKey)

	return &encryptionServiceImpl{
		db:        db,
		masterKey: key,
	}
}

// deriveKey 使用 PBKDF2 从主密钥派生 32 字节的 AES-256 密钥
func deriveKey(masterKey string) []byte {
	// 使用固定 salt（生产环境建议使用配置的 salt）
	salt := []byte("devops-encryption-salt-v1")
	// 迭代次数：OWASP 推荐至少 600,000 次，这里使用 100,000 作为平衡
	iterations := 100000
	keyLen := 32 // AES-256

	return pbkdf2.Key([]byte(masterKey), salt, iterations, keyLen, sha256.New)
}

func (s *encryptionServiceImpl) Encrypt(ctx context.Context, plaintext []byte) (string, error) {
	// 获取或创建数据密钥
	dataKey, err := s.getDataKey(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// 使用 AES-256-GCM 加密
	block, err := aes.NewCipher(dataKey)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *encryptionServiceImpl) Decrypt(ctx context.Context, ciphertext string) ([]byte, error) {
	// 解码 base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCiphertext, err)
	}

	// 获取数据密钥
	dataKey, err := s.getDataKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	// 使用 AES-256-GCM 解密
	block, err := aes.NewCipher(dataKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

func (s *encryptionServiceImpl) EncryptString(ctx context.Context, plaintext string) (string, error) {
	return s.Encrypt(ctx, []byte(plaintext))
}

func (s *encryptionServiceImpl) DecryptString(ctx context.Context, ciphertext string) (string, error) {
	plaintext, err := s.Decrypt(ctx, ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *encryptionServiceImpl) GetOrCreateKey(ctx context.Context) (*models.EncryptionKey, error) {
	var key models.EncryptionKey

	// 查找现有密钥（全局密钥）
	err := s.db.WithContext(ctx).
		Where("status = ?", "active").
		First(&key).Error

	if err == nil {
		return &key, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("查询密钥失败: %w", err)
	}

	// 创建新密钥
	dataKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dataKey); err != nil {
		return nil, fmt.Errorf("生成密钥失败: %w", err)
	}

	// 使用主密钥加密数据密钥
	encryptedKey, err := s.encryptWithMasterKey(dataKey)
	if err != nil {
		return nil, fmt.Errorf("加密密钥失败: %w", err)
	}

	key = models.EncryptionKey{
		KeyID:        "global-key-v1",
		EncryptedKey: encryptedKey,
		Algorithm:    "AES-256-GCM",
		Status:       "active",
		Version:      1,
	}

	if err := s.db.WithContext(ctx).Create(&key).Error; err != nil {
		return nil, fmt.Errorf("保存密钥失败: %w", err)
	}

	// 缓存解密后的密钥
	s.keyCache.Store("global", dataKey)

	return &key, nil
}

func (s *encryptionServiceImpl) RotateKey(ctx context.Context) error {
	// 获取当前密钥
	var oldKey models.EncryptionKey
	if err := s.db.WithContext(ctx).
		Where("status = ?", "active").
		First(&oldKey).Error; err != nil {
		return fmt.Errorf("获取当前密钥失败: %w", err)
	}

	// 生成新密钥
	newDataKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newDataKey); err != nil {
		return fmt.Errorf("生成新密钥失败: %w", err)
	}

	encryptedKey, err := s.encryptWithMasterKey(newDataKey)
	if err != nil {
		return fmt.Errorf("加密新密钥失败: %w", err)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将旧密钥标记为 retired
		if err := tx.Model(&models.EncryptionKey{}).
			Where("id = ?", oldKey.ID).
			Update("status", "retired").Error; err != nil {
			return err
		}

		// 创建新密钥
		newKey := models.EncryptionKey{
			KeyID:        fmt.Sprintf("global-key-v%d", oldKey.Version+1),
			EncryptedKey: encryptedKey,
			Algorithm:    "AES-256-GCM",
			Status:       "active",
			Version:      oldKey.Version + 1,
		}

		if err := tx.Create(&newKey).Error; err != nil {
			return err
		}

		// 更新缓存
		s.keyCache.Store("global", newDataKey)

		return nil
	})
}

// getDataKey 获取解密后的数据密钥
func (s *encryptionServiceImpl) getDataKey(ctx context.Context) ([]byte, error) {
	// 先从缓存获取
	if cached, ok := s.keyCache.Load("global"); ok {
		return cached.([]byte), nil
	}

	// 从数据库获取
	key, err := s.GetOrCreateKey(ctx)
	if err != nil {
		return nil, err
	}

	// 解密数据密钥
	dataKey, err := s.decryptWithMasterKey(key.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("解密数据密钥失败: %w", err)
	}

	// 缓存
	s.keyCache.Store("global", dataKey)

	return dataKey, nil
}

// encryptWithMasterKey 使用主密钥加密
func (s *encryptionServiceImpl) encryptWithMasterKey(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decryptWithMasterKey 使用主密钥解密
func (s *encryptionServiceImpl) decryptWithMasterKey(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("密文太短")
	}

	nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertextBytes, nil)
}
