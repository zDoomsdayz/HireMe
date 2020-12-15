package security

import (
	"fmt"
	"testing"

	. "github.com/franela/goblin"
)

func TestSecurity(t *testing.T) {
	gob := Goblin(t)

	gob.Describe("Security File Test", func() {
		gob.It("should check for ASCII", func() {
			gob.Assert(IsASCII("¥¶»φ")).Equal(false)
			gob.Assert(IsASCII("abcd")).Equal(true)
		})

		gob.It("should check for valid password", func() {
			gob.Assert(CheckPassword("¥¶»φ")).Equal(fmt.Errorf("password only accept Ascii"))
			gob.Assert(CheckPassword("abc")).Equal(fmt.Errorf("minimum password length of 10 or more characters"))
			gob.Assert(CheckPassword("abcefghijkl")).Equal(fmt.Errorf("password need to have at least 1 number"))
			gob.Assert(CheckPassword("123efghijkl")).Equal(fmt.Errorf("password need to have at least 1 uppercase"))
			gob.Assert(CheckPassword("123EFGHIJKL")).Equal(fmt.Errorf("password need to have at least 1 lowercase"))
			gob.Assert(CheckPassword("123efghIJKL")).Equal(fmt.Errorf("password need to have at least 1 symbol"))
			gob.Assert(CheckPassword("123efghIJKL!@#")).Equal(nil)
		})

		gob.It("should check for valid email", func() {
			gob.Assert(CheckEmail("abc")).Equal(fmt.Errorf("E-mail is not valid"))
			gob.Assert(CheckEmail("abc@asd")).Equal(fmt.Errorf("E-mail is not valid"))
			gob.Assert(CheckEmail("abc@asd@asd")).Equal(fmt.Errorf("E-mail is not valid"))
			gob.Assert(CheckEmail("abc@ .com")).Equal(fmt.Errorf("E-mail is not valid"))
			gob.Assert(CheckEmail("a b c@ .com")).Equal(fmt.Errorf("E-mail is not valid"))
			gob.Assert(CheckEmail("a\"bc@asda.com")).Equal(fmt.Errorf("E-mail is not valid"))
			gob.Assert(CheckEmail("abc@asda.com")).Equal(nil)
		})

		gob.It("should encrypt and decrypt message", func() {
			encryptedMessage, _ := Encrypt([]byte("one"), "")
			decryptedMessage, _ := Decrypt(encryptedMessage, "")
			gob.Assert(string(decryptedMessage)).Equal("one")
		})

		gob.It("should hash a password", func() {
			pw1, _ := HashPassword("abc123", "")
			pw2, _ := HashPassword("123asd", "123")
			gob.Assert(HashPasswordCompare([]byte("abc123"), "", pw1)).Equal(nil)
			gob.Assert(HashPasswordCompare([]byte("123asd"), "123", pw2)).Equal(nil)
			gob.Assert(HashPasswordCompare([]byte("123asd"), "123", pw1)).IsNotNil()
		})
	})
}
