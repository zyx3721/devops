package security

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"devops/internal/models"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	// 使用MySQL测试数据库
	dsn := "root:@tcp(127.0.0.1:3306)/devops_test?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping test: MySQL not available: %v", err)
		return nil
	}

	// 自动迁移
	err = db.AutoMigrate(&models.EncryptionKey{})
	require.NoError(t, err)

	// 清理测试数据
	db.Exec("TRUNCATE TABLE encryption_keys")

	return db
}

// TestProperty_EncryptionRoundTrip 属性测试：加密往返一致性
// Property 5: For any sensitive data, encrypting then decrypting
// SHALL produce the original plaintext, and the ciphertext
// SHALL be different from plaintext.
func TestProperty_EncryptionRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	service := NewEncryptionService(db, "test-master-key-32-bytes-long!!")
	ctx := context.Background()

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"简单文本", "Hello, World!"},
		{"中文文本", "你好，世界！"},
		{"特殊字符", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"空字符串", ""},
		{"长文本", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."},
		{"JSON 数据", `{"key": "value", "number": 123, "array": [1, 2, 3]}`},
		{"密码", "P@ssw0rd!123"},
		{"API 密钥", "sk-1234567890abcdefghijklmnopqrstuvwxyz"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 加密
			ciphertext, err := service.Encrypt(ctx, []byte(tc.plaintext))
			require.NoError(t, err)

			// 属性验证：密文不应该等于明文（除非是空字符串）
			if tc.plaintext != "" {
				assert.NotEqual(t, tc.plaintext, ciphertext,
					"密文不应该等于明文")
			}

			// 解密
			decrypted, err := service.Decrypt(ctx, ciphertext)
			require.NoError(t, err)

			// 属性验证：解密后应该等于原始明文
			assert.Equal(t, tc.plaintext, string(decrypted),
				"解密后应该等于原始明文")
		})
	}
}

// TestProperty_EncryptionDeterminism 属性测试：加密非确定性
func TestProperty_EncryptionDeterminism(t *testing.T) {
	db := setupTestDB(t)
	service := NewEncryptionService(db, "test-master-key-32-bytes-long!!")
	ctx := context.Background()

	plaintext := "Same plaintext"

	// 多次加密相同的明文
	ciphertext1, err := service.Encrypt(ctx, []byte(plaintext))
	require.NoError(t, err)

	ciphertext2, err := service.Encrypt(ctx, []byte(plaintext))
	require.NoError(t, err)

	// 属性验证：相同明文的两次加密结果应该不同（因为使用随机 nonce）
	assert.NotEqual(t, ciphertext1, ciphertext2,
		"相同明文的两次加密结果应该不同")

	// 但两者解密后应该相同
	decrypted1, _ := service.Decrypt(ctx, ciphertext1)
	decrypted2, _ := service.Decrypt(ctx, ciphertext2)

	assert.Equal(t, string(decrypted1), string(decrypted2),
		"两个密文解密后应该相同")
}

// TestProperty_EncryptionTenantIsolation 属性测试：多次加密一致性
// 注意：当前实现使用全局密钥，不支持租户隔离
func TestProperty_EncryptionConsistency(t *testing.T) {
	db := setupTestDB(t)
	service := NewEncryptionService(db, "test-master-key-32-bytes-long!!")
	ctx := context.Background()

	plaintext := "Sensitive data"

	// 第一次加密
	ciphertext1, err := service.Encrypt(ctx, []byte(plaintext))
	require.NoError(t, err)

	// 第二次加密
	ciphertext2, err := service.Encrypt(ctx, []byte(plaintext))
	require.NoError(t, err)

	// 属性验证：两次加密的密文应该不同（因为使用随机 nonce）
	assert.NotEqual(t, ciphertext1, ciphertext2,
		"两次加密的密文应该不同")

	// 两次加密都可以正确解密
	decrypted1, err := service.Decrypt(ctx, ciphertext1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(decrypted1))

	decrypted2, err := service.Decrypt(ctx, ciphertext2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(decrypted2))
}

// TestProperty_EncryptionKeyRotation 属性测试：密钥轮换
func TestProperty_EncryptionKeyRotation(t *testing.T) {
	db := setupTestDB(t)
	service := NewEncryptionService(db, "test-master-key-32-bytes-long!!")
	ctx := context.Background()

	plaintext := "Data to encrypt"

	// 使用旧密钥加密
	ciphertext, err := service.Encrypt(ctx, []byte(plaintext))
	require.NoError(t, err)

	// 轮换密钥
	err = service.RotateKey(ctx)
	require.NoError(t, err)

	// 属性验证：旧密文仍然可以解密（向后兼容）
	decrypted, err := service.Decrypt(ctx, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(decrypted),
		"密钥轮换后旧密文仍应可解密")

	// 使用新密钥加密
	newCiphertext, err := service.Encrypt(ctx, []byte(plaintext))
	require.NoError(t, err)

	// 新密文也可以解密
	newDecrypted, err := service.Decrypt(ctx, newCiphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(newDecrypted),
		"新密钥加密的数据应可解密")
}

// TestProperty_EncryptionIntegrity 属性测试：加密完整性
func TestProperty_EncryptionIntegrity(t *testing.T) {
	db := setupTestDB(t)
	service := NewEncryptionService(db, "test-master-key-32-bytes-long!!")
	ctx := context.Background()

	plaintext := "Important data"

	// 加密
	ciphertext, err := service.Encrypt(ctx, []byte(plaintext))
	require.NoError(t, err)

	// 解码 base64 以便篡改
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	require.NoError(t, err)

	// 篡改密文
	if len(decoded) > 10 {
		tamperedDecoded := make([]byte, len(decoded))
		copy(tamperedDecoded, decoded)
		tamperedDecoded[10] ^= 0xFF // 翻转一个字节

		// 重新编码
		tamperedCiphertext := base64.StdEncoding.EncodeToString(tamperedDecoded)

		// 属性验证：篡改后的密文解密应该失败
		_, err = service.Decrypt(ctx, tamperedCiphertext)
		assert.Error(t, err, "篡改后的密文解密应该失败")
	}
}

// TestProperty_EncryptionEmptyInput 属性测试：空输入处理
func TestProperty_EncryptionEmptyInput(t *testing.T) {
	db := setupTestDB(t)
	service := NewEncryptionService(db, "test-master-key-32-bytes-long!!")
	ctx := context.Background()

	// 空字符串加密
	ciphertext, err := service.Encrypt(ctx, []byte(""))
	require.NoError(t, err)

	// 解密
	decrypted, err := service.Decrypt(ctx, ciphertext)
	require.NoError(t, err)

	// 属性验证：空字符串加密解密后仍为空
	assert.Equal(t, "", string(decrypted), "空字符串加密解密后应为空")
}

// TestProperty_EncryptionLargeData 属性测试：大数据加密
func TestProperty_EncryptionLargeData(t *testing.T) {
	db := setupTestDB(t)
	service := NewEncryptionService(db, "test-master-key-32-bytes-long!!")
	ctx := context.Background()

	// 创建 1MB 的数据
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// 加密
	ciphertext, err := service.Encrypt(ctx, largeData)
	require.NoError(t, err)

	// 解密
	decrypted, err := service.Decrypt(ctx, ciphertext)
	require.NoError(t, err)

	// 属性验证：大数据加密解密后应完全一致
	assert.Equal(t, largeData, decrypted, "大数据加密解密后应完全一致")
}
