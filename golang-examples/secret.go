package main

import "fmt"
import "encoding/base64"

func main() {
	fmt.Println("secrets...")
	enUser, enPass := Encode("testuser", "password")
	fmt.Printf("%s %s\n", enUser, enPass)
	deUser, dePass := Decode(enUser, enPass)
	fmt.Printf("%s %s\n", deUser, dePass)
	deUser, dePass = Decode("cGd1c2VyMQ==", "cGFzc3dvcmQ=")
	fmt.Printf("%s %s\n", deUser, dePass)
}

func Encode(username string, password string) (string, string) {
	return base64.StdEncoding.EncodeToString([]byte(username)), base64.StdEncoding.EncodeToString([]byte(password))

}

func Decode(username string, password string) ([]byte, []byte) {
	var err error
	var deUser, dePass []byte

	deUser, err = base64.StdEncoding.DecodeString(username)
	dePass, err = base64.StdEncoding.DecodeString(password)
	if err != nil {
		fmt.Println(err.Error())
	}
	return deUser, dePass

}
