package password

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestNewSCRAMPassword(t *testing.T) {
	password := "datalake"

	scram := NewSCRAMPassword(password)

	if scram.password != password {
		t.Errorf("plaintext password not set properly. expected %q actual %q", password, scram.password)
		return
	}

	if scram.Iterations != scramDefaultIterations {
		t.Errorf("iterations not set properly. expected %d actual %d", scramDefaultIterations, scram.Iterations)
		return
	}

	if scram.SaltLength != scramDefaultSaltLength {
		t.Errorf("salt length not set properly. expected %d actual %d", scramDefaultSaltLength, scram.SaltLength)
		return
	}
}

func TestScramGenerateSalt(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		saltLengths := []int{
			scramDefaultSaltLength,
			scramDefaultSaltLength * 2,
		}

		for _, saltLength := range saltLengths {
			t.Run(fmt.Sprintf("salt length %d", saltLength), func(t *testing.T) {
				salt, err := scramGenerateSalt(saltLength)
				if err != nil {
					t.Error(err)
				}

				if len(salt) != saltLength {
					t.Errorf("expected: %d actual: %d", saltLength, len(salt))
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		saltLengths := []int{0, -1}

		for _, saltLength := range saltLengths {
			t.Run(fmt.Sprintf("salt length %d", saltLength), func(t *testing.T) {
				if _, err := scramGenerateSalt(saltLength); err == nil {
					t.Errorf("error expected for salt length of %d", saltLength)
				}
			})
		}
	})
}

func TestSCRAMBuild(t *testing.T) {
	t.Run("scram-sha-256", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			// check a few different password combinations. note: the salt is kept the
			// same so we can get a reproducible result
			credentialList := []([]string){
				[]string{`datalake`, `SCRAM-SHA-256$4096:aDFwcDBwNHJ0eTIwMjA=$xHkOo65LX9eBB8a6v+axqvs3+aMBTH0sCT7w/Nxzh5M=:PXuFoeJNuAGSeExskYSqkwUyiUJu8LPC9DgwDWQ9ARQ=`},
				[]string{`øásis`, `SCRAM-SHA-256$4096:aDFwcDBwNHJ0eTIwMjA=$ySGUcYGGJXsigb0a24AfSqNRpGM+zqwlkfuzdlWCV9k=:GDITAfQzF7M9aJaP5OK04b6bT+XQ+wjU3qiGC2ERxeA=`},
				[]string{`md53a0689aa9e31a50b5621971fc89f0c64`, `SCRAM-SHA-256$4096:aDFwcDBwNHJ0eTIwMjA=$R93U562i0T1ewqfMD3JhD/eTnvTsVBDq1wzkBWx0+WU=:p+dt112MXgpsvAshbNU6jTSMegApKRzb9VT18yiQ/HY=`},
				[]string{`SCRAM-SHA-256$4096:aDFwcDBwNHJ0eTIwMjA=$xHkOo65LX9eBB8a6v+axqvs3+aMBTH0sCT7w/Nxzh5M=:PXuFoeJNuAGSeExskYSqkwUyiUJu8LPC9DgwDWQ9ARQ=`, `SCRAM-SHA-256$4096:aDFwcDBwNHJ0eTIwMjA=$s9HbNQBsfJwflGr4lvr4vEt/vvspp5Uu8IjWYLjMUMg=:3sUGJgo/70EQvjsma2I/RJsheqLhxN2rarUt7oqK6q8=`},
			}
			mockGenerateSalt := func(length int) ([]byte, error) {
				// return the special salt
				return []byte("h1pp0p4rty2020"), nil
			}

			// a crednetial is valid if it generates the specified md5 hash
			for _, credentials := range credentialList {
				t.Run(credentials[0], func(t *testing.T) {
					scram := NewSCRAMPassword(credentials[0])
					scram.generateSalt = mockGenerateSalt

					hash, err := scram.Build()
					if err != nil {
						t.Error(err)
					}

					if hash != credentials[1] {
						t.Errorf("expected: %q actual %q", credentials[1], hash)
					}
				})
			}
		})

		t.Run("invalid", func(t *testing.T) {
			// ensure the generate salt function returns an error
			mockGenerateSalt := func(length int) ([]byte, error) {
				return []byte{}, ErrSCRAMSaltLengthInvalid
			}

			t.Run("invalid salt generator value", func(t *testing.T) {
				scram := NewSCRAMPassword("datalake")
				scram.generateSalt = mockGenerateSalt

				if _, err := scram.Build(); err == nil {
					t.Errorf("error expected with invalid value to salt generator")
				}
			})
		})
	})
}

func TestSCRAMEncode(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		scram := SCRAMPassword{}
		expected := "aGlwcG8="
		actual := scram.encode([]byte("hippo"))

		if expected != actual {
			t.Errorf("expected: %s actual %s", expected, actual)
		}
	})
}

func TestSCRAMHash(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		scram := SCRAMPassword{}
		expected, _ := hex.DecodeString("877cc977e7b033e10d6e0b0d666da1f463bc51b1de48869250a0347ec1b2b8b3")
		actual := scram.hash(sha256.New, []byte("hippo"))

		if !bytes.Equal(expected, actual) {
			t.Errorf("expected: %x actual %x", expected, actual)
		}
	})
}

func TestSCRAMHMAC(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		scram := SCRAMPassword{}
		expected, _ := hex.DecodeString("ac9872eb21043142c3bf073c9fa4caf9553940750ef7b85116905aaa456a2d07")
		actual := scram.hmac(sha256.New, []byte("hippo"), []byte("datalake"))

		if !bytes.Equal(expected, actual) {
			t.Errorf("expected: %x actual %x", expected, actual)
		}
	})
}

func TestSCRAMIsASCII(t *testing.T) {
	type stringStruct struct {
		str     string
		isASCII bool
	}

	tests := []stringStruct{
		{str: "hippo", isASCII: true},
		{str: "híppo", isASCII: false},
		{str: "こんにちは", isASCII: false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("is ascii %q", test.str), func(t *testing.T) {
			scram := SCRAMPassword{password: test.str}

			if scram.isASCII() != test.isASCII {
				t.Errorf("%q should be %t", test.str, test.isASCII)
			}
		})
	}
}

func TestSCRAMSASLPrep(t *testing.T) {
	type stringStruct struct {
		password string
		expected string
	}

	// some of the testing methodology for this is borrowed from:
	//
	// https://github.com/MagicStack/asyncpg/blob/master/tests/test_connect.py#L276-L287
	tests := []stringStruct{
		{password: "hippo", expected: "hippo"},
		{password: "híppo", expected: "híppo"},
		{password: "こんにちは", expected: "こんにちは"},
		{password: "hippo\u1680lake", expected: "hippo lake"},
		{password: "hipp\ufe01o", expected: "hippo"},
		{password: "hipp\u206ao", expected: "hipp\u206ao"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("saslprep %q", test.password), func(t *testing.T) {
			scram := SCRAMPassword{password: test.password}

			if scram.saslPrep() != test.expected {
				t.Errorf("%q should be %q", test.password, test.expected)
			}
		})
	}
}
