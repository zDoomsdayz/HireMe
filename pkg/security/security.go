// Package security handles all the software development techniques I need for this assigment.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// Encrypt uses Md5 and GCM to seal the data and return it as a byte and error.
func Encrypt(data []byte, customKey string) ([]byte, error) {
	key := md5.New()
	if customKey == "" {
		key.Write([]byte("default"))
	} else {
		key.Write([]byte(customKey))
	}

	block, err := aes.NewCipher(key.Sum(nil))
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	encryptedData := aesgcm.Seal(nonce, nonce, data, nil)
	return encryptedData, nil
}

// Decrypt uses Md5 and GCM to open the data and return it as a byte and error.
func Decrypt(data []byte, customKey string) ([]byte, error) {
	key := md5.New()
	if customKey == "" {
		key.Write([]byte("default"))
	} else {
		key.Write([]byte(customKey))
	}

	block, err := aes.NewCipher(key.Sum(nil))
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesgcm.NonceSize()
	nonce, text := data[:nonceSize], data[nonceSize:]

	decryptedData, err := aesgcm.Open(nil, nonce, text, nil)
	if err != nil {
		return nil, err
	}
	return decryptedData, nil
}

// HashPassword uses bcrypt, sha512, pepper and use encrypt for another layer for protection
func HashPassword(password, pepper string) ([]byte, error) {
	hash := sha512.New()
	hash.Write([]byte(password))

	hashPass, err := bcrypt.GenerateFromPassword(hash.Sum(nil), 10)
	if err != nil {
		return nil, err
	}
	encryptHash, err := Encrypt(hashPass, pepper)

	return encryptHash, nil
}

// HashPasswordCompare decrypt first then sha512 the password and then compare with bcrypt
func HashPasswordCompare(password []byte, pepper string, encryptedHash []byte) error {
	decryptHash, err := Decrypt(encryptedHash, pepper)
	if err != nil {
		return err
	}
	hash := sha512.New()
	hash.Write(password)

	err = bcrypt.CompareHashAndPassword(decryptHash, hash.Sum(nil))
	if err != nil {
		return err
	}

	return nil
}

// IsASCII will loop though the string to check for ASCII and return as bool
func IsASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// CheckPassword checks if the password contains any ASCII, length, lower/ uppercase and symbol
func CheckPassword(password string) error {

	if !IsASCII(password) {
		return fmt.Errorf("password only accept Ascii")
	}

	if len(password) < 10 {
		return fmt.Errorf("minimum password length of 10 or more characters")
	}

	if !strings.ContainsAny(password, "1234567890") {
		return fmt.Errorf("password need to have at least 1 number")
	}

	if !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		return fmt.Errorf("password need to have at least 1 lowercase")
	}

	if !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return fmt.Errorf("password need to have at least 1 uppercase")
	}

	if !strings.ContainsAny(password, "~!@#$%^&*()-+|_") {
		return fmt.Errorf("password need to have at least 1 symbol")
	}
	return nil
}

// CheckEmail uses regexp to check if it's a valid email
func CheckEmail(email string) error {
	Re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

	if !Re.MatchString(email) {
		return fmt.Errorf("E-mail is not valid")
	}

	return nil
}
