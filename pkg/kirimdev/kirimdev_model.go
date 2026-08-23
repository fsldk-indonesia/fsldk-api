// Package kirimdev memuat entitas klien WhatsApp Kirimdev. Seluruhnya murni
// struct data (tanpa function/method) — logika pengiriman berada di kirimdev.go.
package kirimdev

// TemplateMessage adalah payload pengiriman satu pesan template WhatsApp.
type TemplateMessage struct {
	ToPhone      string // format E.164 tanpa "+", mis. "6281234567890"
	TemplateName string // nama template terdaftar di Kirimdev/Meta, mis. "shortlink_request_notice"
	Language     string
	Params       []string // parameter posisional template BODY, urut sesuai definisi template
	// ButtonPayloads, kalau diisi, mengirim payload custom per tombol
	// QUICK_REPLY template ini (urut sesuai definisi tombol saat template
	// dibuat, index 0 = tombol pertama, dst) — payload inilah yang nanti
	// muncul di InboundButton.Payload saat penerima menekan tombol (§1a.5
	// techspec). Kosong = template ini tidak punya tombol.
	ButtonPayloads []string
}

// SendResult adalah hasil pengiriman satu pesan template WhatsApp. MessageID
// adalah ID internal Kirimdev ("msg_..."), BUKAN wamid Meta — respons sinkron
// Kirimdev tidak pernah membawa wamid (dikoreksi 2026-08-23 dari observasi
// live, lihat kirimdev.go). Ini yang dicatat ke tr_whatsapp_message_log
// sebagai kunci korelasi awal; wamid ASLI baru menyusul lewat event webhook
// "message.sent" dan dipakai memperbarui baris yang sama (§1a.5/§1b techspec,
// lihat jobqueue_service.HandleMessageSent) supaya context.id balasan PIC
// (yang selalu berisi wamid asli, bukan ID Kirimdev) tetap bisa dicocokkan.
type SendResult struct {
	MessageID string
	Status    string
}

// InboundWebhookPayload adalah SATU pesan WhatsApp masuk (balasan PIC) yang
// sudah diekstrak & diratakan dari amplop webhook Kirimdev — bentuk asli di
// atas kawat mengikuti format WhatsApp Cloud API Meta apa adanya (Kirimdev
// meneruskan payload byte-for-byte, lihat docs.kirimdev.com), dibungkus
// entry[].changes[].value.messages[]; parsing amplop tsb + flatten jadi
// struct ini adalah tanggung jawab Client.ParseInboundWebhook (kirimdev.go).
// Kalau amplop tidak berisi pesan (mis. event status/delivery, bukan
// message.received), From/Type kosong — pemanggil (HandleWhatsAppReply)
// sudah menangani ini sebagai "tidak ada intent terdeteksi", bukan error.
type InboundWebhookPayload struct {
	From        string              // nomor pengirim, format E.164 tanpa "+"
	MessageID   string              // wamid pesan inbound ini
	Type        string              // "text" | "button" | "interactive"
	Text        *InboundText        // isi kalau Type == "text"
	Button      *InboundButton      // isi kalau Type == "button" (quick-reply template)
	Interactive *InboundInteractive // isi kalau Type == "interactive"
	Context     *InboundContext     // ada kalau ini reply/quote ke pesan lain
}

// InboundText adalah isi balasan bertipe teks bebas (mis. "YES"/"NO").
type InboundText struct {
	Body string
}

// InboundButton adalah isi balasan lewat tombol quick-reply template.
type InboundButton struct {
	Payload string // mis. "approve" / "reject"
	Text    string
}

// InboundInteractive adalah isi balasan lewat komponen interaktif (mis. list/button Meta).
type InboundInteractive struct {
	Type        string // "button_reply"
	ButtonReply *InboundButtonReply
}

// InboundButtonReply adalah detail tombol yang ditekan pada InboundInteractive.
type InboundButtonReply struct {
	ID    string
	Title string
}

// InboundContext menunjuk pesan yang di-reply/quote oleh balasan ini — dipakai
// resolusi "balasan ini soal request yang mana" lewat tr_whatsapp_message_log
// (§1a.5/§1b techspec), bukan cache pointer per-nomor seperti referensi Laravel.
type InboundContext struct {
	ID string // wamid pesan yang di-reply
}

// DeliveryStatus adalah SATU status pengiriman dari event webhook
// "message.status" Kirimdev (§12 techspec) — Meta/Kirimdev membalas
// pengiriman template dengan 200 duluan (status "pending"/"queued"), lalu
// kegagalan sesungguhnya (mis. template_not_found) baru diketahui belakangan
// lewat event async ini, BUKAN dari respons SendTemplate. Diekstrak &
// diratakan dari amplop Meta apa adanya oleh Client.ParseDeliveryStatusWebhook.
type DeliveryStatus struct {
	WAMessageID string // wamid (atau ID internal Kirimdev) pesan yang statusnya berubah
	Status      string // "sent" | "delivered" | "read" | "failed" | "played"
	ErrorDetail string // terisi kalau Status == "failed" — gabungan title+message+code dari Meta
}
