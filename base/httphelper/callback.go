package httphelper

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
)

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
	return false, json.Unmarshal(body, obj)
}
