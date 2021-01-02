package pgadmin

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"testing"
)

var testData = struct {
	clearPW string
	encPW   string
	key     string
	iv      []byte
}{
	clearPW: "w052H0UBM783B$x6N___",
	encPW:   "5PN+lp8XXalwRzCptI21hmT5S9FvvEYpD8chWa39akY6Srwl",
	key:     "$pbkdf2-sha512$19000$knLuvReC8H7v/T8n5JwTwg$OsVGpDa/zpCE2pKEOsZ4/SqdxcQZ0UU6v41ev/gkk4ROsrws/4I03oHqN37k.v1d25QckESs3NlPxIUv5gTf2Q",
	iv: []byte{
		0xe4, 0xf3, 0x7e, 0x96, 0x9f, 0x17, 0x5d, 0xa9,
		0x70, 0x47, 0x30, 0xa9, 0xb4, 0x8d, 0xb5, 0x86,
	},
}

func TestSymmetry(t *testing.T) {
	expected := "Hello World! How are you today?"
	ciphertext := encrypt(expected, testData.key)
	decoded := decrypt(ciphertext, testData.key)
	if decoded != expected {
		t.Fatalf("\nExpected\t[%s]\nReceived\t[%s]\n", expected, decoded)
	}
}

func TestEncryption(t *testing.T) {
	encrypted := encryptImpl(testData.key, []byte(testData.clearPW), testData.iv)
	if encrypted != testData.encPW {
		t.Fatalf("\nExpected\t[%s]\nReceived\t[%s]\n", testData.encPW, encrypted)
	}
}

func TestDecryption(t *testing.T) {
	decrypted := decrypt(testData.encPW, testData.key)

	if decrypted != testData.clearPW {
		t.Fatalf("\nExpected\t[%s]\nReceived\t[%s]\n", testData.clearPW, decrypted)
	}
}

func TestShortKey(t *testing.T) {
	expected := "JwTwg$OsVG}}}}}}}}}}}}}}}}}}}}}}"
	paddedKey := padKey([]byte("JwTwg$OsVG"))
	if string(paddedKey) != expected {
		t.Fatalf("\nExpected\t[%s]\nReceived\t[%s]\n", expected, paddedKey)
	}
}

func TestSymmetryShortKey(t *testing.T) {
	expected := "Hello World! How are you today?"
	ciphertext := encrypt(expected, "JwTwg$OsVG")
	decoded := decrypt(ciphertext, "JwTwg$OsVG")
	if decoded != expected {
		t.Fatalf("\nExpected\t[%s]\nReceived\t[%s]\n", expected, decoded)
	}
}
