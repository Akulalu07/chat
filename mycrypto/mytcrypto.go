package mycrypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
)

type Loger struct {
	time uint64
	text string
	key  string
}

var synbols = []byte("poiuytrewqzxcvbnmlkjhgfdsa")

var localkey string

func Generate_salt() string {
	// 16 символов

	somestring := []byte("")
	for i := 0; i < 16; i++ {
		x := rand.Int31n(16)
		somestring = append(somestring, synbols[x])
	}
	return fmt.Sprintf("%s", somestring)

}
func Init() {
	data, err := os.ReadFile("key.txt")
	if err != nil {
		log.Fatal(err)
	}
	localkey = fmt.Sprintf("%s", data)
}

func PasswordToHash(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(password + salt))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func textToTextWithKey(mess Loger) string {
	mac := hmac.New(sha256.New, []byte(localkey))
	mac.Write([]byte(mess.text + fmt.Sprintf("%d", mess.time)))
	return string(mac.Sum(nil))
}

func Sertificate(mess Loger, timeel uint64) bool {
	if mess.time+timeel > uint64(time.Now().Unix()) {
		return false
	}
	key := textToTextWithKey(mess)
	return key == mess.key
}
