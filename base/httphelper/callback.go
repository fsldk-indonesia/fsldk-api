package httphelper

import (
	"bytes"
	"encoding/json"
	"io"
	"log"

	"github.com/gin-gonic/gin"
)

// maxLoggedCallbackBodyBytes membatasi panjang body yang di-log saat parsing
// gagal — cukup untuk mendiagnosis bentuk payload gateway pihak ketiga tanpa
// membanjiri log dengan body yang sangat besar/tidak wajar.
const maxLoggedCallbackBodyBytes = 1000

// BindCallbackJSON mem-parse body JSON webhook callback pihak ketiga
// (Bisatopup payment/disbursement), dengan toleransi khusus: body benar-benar
// kosong dianggap "connectivity ping" — bukan error. Dashboard sebagian
// payment gateway (termasuk Bisatopup, tombol "Test" pada field Url
// Callback) mengirim request tanpa body sama sekali sekadar mengecek
// endpoint hidup & membalas 2xx, BUKAN simulasi payload webhook asli
// (payload asli selalu berisi field transaksi). Sebelum toleransi ini,
// ping semacam itu selalu dibalas 400 "Format callback tidak valid" karena
// gagal di-decode sebagai JSON, padahal integrasinya sendiri baik-baik saja.
//
// Body yang berisi sesuatu tapi BUKAN JSON valid tetap dianggap error
// seperti biasa (isPing=false, err != nil) — toleransi ini murni untuk body
// kosong, bukan untuk payload yang salah format.
func BindCallbackJSON(c *gin.Context, obj interface{}) (isPing bool, err error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return true, nil
	}
	if err := json.Unmarshal(body, obj); err != nil {
		// Body non-kosong tapi gagal di-parse — kemungkinan gateway pihak
		// ketiga mengirim bentuk payload yang belum kita duga (bukan JSON
		// object, Content-Type bukan JSON, dsb). Di-log supaya bisa
		// didiagnosis dari body mentahnya, bukan cuma pesan error generik
		// "Format callback tidak valid" yang dilihat pengirimnya.
		logged := body
		if len(logged) > maxLoggedCallbackBodyBytes {
			logged = logged[:maxLoggedCallbackBodyBytes]
		}
		log.Printf("[CALLBACK] gagal parse body dari %s %s: %v — body mentah (maks %d byte): %s",
			c.Request.Method, c.Request.URL.Path, err, maxLoggedCallbackBodyBytes, string(logged))
		return false, err
	}
	return false, nil
}
