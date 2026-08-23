// Package kirimdev mengirim & menerima pesan WhatsApp lewat Kirimdev
// (api.kirimdev.com — official Meta WhatsApp Business Cloud API passthrough).
//
// Shape request/response di bawah mengikuti docs.kirimdev.com (diverifikasi
// 2026-08-21): endpoint kirim pesan adalah POST {baseURL}/{phoneNumberID}/messages
// dengan body persis payload WhatsApp Cloud API Meta (diteruskan byte-for-byte
// oleh Kirimdev, bukan di-rewrite), signature webhook satu header
// "X-Kirim-Signature: t=<unix>,v1=<hex>[,v1=<hex>...]" (bukan header terpisah
// untuk timestamp), dan amplop webhook inbound mengikuti format Meta
// entry[].changes[].value.messages[] apa adanya.
package kirimdev

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client adalah klien pengiriman & penerimaan pesan WhatsApp via Kirimdev.
type Client struct {
	apiKey         string
	phoneNumberID  string
	baseURL        string
	language       string
	webhookSecrets []string
	replyWindow    time.Duration
}

// NewClient membuat Client Kirimdev. replyWindow adalah toleransi replay
// untuk verifikasi signature webhook (§7 techspec, default 5 menit).
func NewClient(apiKey, phoneNumberID, baseURL, language string, webhookSecrets []string, replyWindow time.Duration) *Client {
	return &Client{
		apiKey: apiKey, phoneNumberID: phoneNumberID, baseURL: strings.TrimRight(baseURL, "/"), language: language,
		webhookSecrets: webhookSecrets, replyWindow: replyWindow,
	}
}

// sendTemplateRequest adalah body persis payload WhatsApp Cloud API Meta
// untuk kirim pesan template (docs.kirimdev.com/sending/send-templates/).
type sendTemplateRequest struct {
	MessagingProduct string           `json:"messaging_product"`
	To               string           `json:"to"`
	Type             string           `json:"type"`
	Template         sendTemplateBody `json:"template"`
}

type sendTemplateBody struct {
	Name       string        `json:"name"`
	Language   string        `json:"language"`
	Components []interface{} `json:"components,omitempty"` // sendBodyComponent dan/atau sendButtonComponent (shape beda per tipe)
}

// sendBodyComponent adalah parameter posisional teks BODY template.
type sendBodyComponent struct {
	Type       string                  `json:"type"`
	Parameters []sendTemplateParameter `json:"parameters"`
}

// sendButtonComponent adalah payload custom untuk SATU tombol QUICK_REPLY
// template (docs.kirimdev.com/sending/send-template-buttons/). Index bukan
// pakai `omitempty` karena tombol pertama = index 0, yang jadi zero-value Go
// dan akan hilang dari JSON kalau di-omitempty.
type sendButtonComponent struct {
	Type       string                  `json:"type"`
	SubType    string                  `json:"sub_type"`
	Index      int                     `json:"index"`
	Parameters []sendTemplateParameter `json:"parameters"`
}

type sendTemplateParameter struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Payload string `json:"payload,omitempty"`
}

// sendTemplateResponse adalah bentuk sukses docs.kirimdev.com/api/operations/phone_number_idmessages/post/.
// Dikoreksi 2026-08-23 dari observasi respons live: field "message_id" (wamid)
// yang didokumentasikan TIDAK PERNAH ada di respons sinkron — cuma "id" (ID
// internal Kirimdev, mis. "msg_...") yang selalu ada. Wamid asli baru
// tersedia belakangan lewat event webhook "message.sent" (data.message.provider_id,
// lihat ParseMessageSentWebhook) — Client.SendTemplate makanya mengembalikan
// "id" ini sebagai SendResult.MessageID, BUKAN wamid.
type sendTemplateResponse struct {
	Data struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"data"`
}

// sendTemplateErrorResponse adalah bentuk error docs.kirimdev.com untuk
// status 400/401/404/422/429/502/503.
type sendTemplateErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Param   string `json:"param"`
	} `json:"error"`
}

// SendTemplate mengirim satu pesan template WhatsApp ke msg.ToPhone.
func (c *Client) SendTemplate(ctx context.Context, msg TemplateMessage) (*SendResult, error) {
	language := msg.Language
	if language == "" {
		language = c.language
	}

	var components []interface{}
	if len(msg.Params) > 0 {
		params := make([]sendTemplateParameter, 0, len(msg.Params))
		for _, p := range msg.Params {
			params = append(params, sendTemplateParameter{Type: "text", Text: p})
		}
		components = append(components, sendBodyComponent{Type: "body", Parameters: params})
	}
	for i, payload := range msg.ButtonPayloads {
		components = append(components, sendButtonComponent{
			Type: "button", SubType: "quick_reply", Index: i,
			Parameters: []sendTemplateParameter{{Type: "payload", Payload: payload}},
		})
	}

	payload, err := json.Marshal(sendTemplateRequest{
		MessagingProduct: "whatsapp",
		To:               msg.ToPhone,
		Type:             "template",
		Template:         sendTemplateBody{Name: msg.TemplateName, Language: language, Components: components},
	})
	if err != nil {
		return nil, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/%s/messages", c.baseURL, c.phoneNumberID)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody sendTemplateErrorResponse
		if json.NewDecoder(resp.Body).Decode(&errBody) == nil && errBody.Error.Message != "" {
			return nil, fmt.Errorf("kirimdev: %s (%s): %s", resp.Status, errBody.Error.Code, errBody.Error.Message)
		}
		return nil, fmt.Errorf("kirimdev: unexpected status %d", resp.StatusCode)
	}

	var body sendTemplateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return &SendResult{MessageID: body.Data.ID, Status: body.Data.Status}, nil
}

// VerifyWebhookSignature memverifikasi header "X-Kirim-Signature" webhook,
// format "t=<unix>,v1=<hex>[,v1=<hex>...]" (docs.kirimdev.com/webhooks/signing/).
// HMAC-SHA256 dihitung atas "{t}.{rawBody}", dicek melawan SETIAP secret di
// c.webhookSecrets dan SETIAP nilai v1 pada header (Kirimdev bisa mengirim
// lebih dari satu v1 saat rotasi secret di sisi mereka) — supaya rotasi
// secret di kedua sisi tetap valid selama masa transisi.
func (c *Client) VerifyWebhookSignature(payload []byte, signatureHeader string) bool {
	var ts int64
	var sigs []string
	for _, part := range strings.Split(signatureHeader, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			parsed, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return false
			}
			ts = parsed
		case "v1":
			sigs = append(sigs, kv[1])
		}
	}
	if ts == 0 || len(sigs) == 0 {
		return false
	}

	age := time.Since(time.Unix(ts, 0))
	if age > c.replyWindow || age < -c.replyWindow {
		return false // di luar replay window (jam terlalu jauh/pesan terlalu tua)
	}

	signedPayload := []byte(strconv.FormatInt(ts, 10) + "." + string(payload))
	for _, secret := range c.webhookSecrets {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(signedPayload)
		expected := hex.EncodeToString(mac.Sum(nil))
		for _, sig := range sigs {
			if hmac.Equal([]byte(expected), []byte(sig)) {
				return true
			}
		}
	}
	return false
}

// webhookEnvelope adalah amplop webhook Kirimdev — mengikuti format WhatsApp
// Cloud API Meta apa adanya (docs.kirimdev.com/webhooks/payloads/), diteruskan
// byte-for-byte oleh Kirimdev.
type webhookEnvelope struct {
	Entry []webhookEntry `json:"entry"`
}

type webhookEntry struct {
	Changes []webhookChange `json:"changes"`
}

type webhookChange struct {
	Value webhookValue `json:"value"`
	Field string       `json:"field"`
}

type webhookValue struct {
	Messages []webhookMessage `json:"messages"`
	Statuses []webhookStatus  `json:"statuses"` // event "message.status" — envelope sama, field beda (§12 techspec)
}

type webhookStatus struct {
	ID     string               `json:"id"` // wamid, atau ID internal Kirimdev (msg_...) untuk pre-delivery failure
	Status string               `json:"status"`
	Errors []webhookStatusError `json:"errors,omitempty"`
}

type webhookStatusError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

type webhookMessage struct {
	From        string                 `json:"from"`
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Context     *webhookMessageContext `json:"context,omitempty"`
	Text        *InboundText           `json:"text,omitempty"`
	Button      *InboundButton         `json:"button,omitempty"`
	Interactive *webhookInteractive    `json:"interactive,omitempty"`
}

type webhookMessageContext struct {
	ID string `json:"id"`
}

// webhookInteractive dipisah dari InboundInteractive (model publik) karena
// nama field Meta yang asli snake_case (button_reply) beda dari konvensi
// camelCase yang dipakai struct data publik package ini.
type webhookInteractive struct {
	Type        string              `json:"type"` // "button_reply"
	ButtonReply *InboundButtonReply `json:"button_reply,omitempty"`
}

// ParseInboundWebhook mem-parsing body webhook Kirimdev (amplop Meta
// entry[].changes[].value.messages[]) dan meratakan PESAN PERTAMA yang
// ditemukan di field "messages" jadi InboundWebhookPayload. Amplop tanpa
// pesan (mis. status/delivery callback, bukan message.received) menghasilkan
// payload kosong — bukan error, pemanggil (HandleWhatsAppReply) sudah
// menangani ini sebagai "tidak ada intent terdeteksi".
func (c *Client) ParseInboundWebhook(body []byte) (InboundWebhookPayload, error) {
	var envelope webhookEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return InboundWebhookPayload{}, err
	}

	for _, entry := range envelope.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" || len(change.Value.Messages) == 0 {
				continue
			}
			msg := change.Value.Messages[0]
			payload := InboundWebhookPayload{
				From: msg.From, MessageID: msg.ID, Type: msg.Type,
				Text: msg.Text, Button: msg.Button,
			}
			if msg.Context != nil {
				payload.Context = &InboundContext{ID: msg.Context.ID}
			}
			if msg.Interactive != nil {
				payload.Interactive = &InboundInteractive{Type: msg.Interactive.Type, ButtonReply: msg.Interactive.ButtonReply}
			}
			return payload, nil
		}
	}
	return InboundWebhookPayload{}, nil
}

// ParseDeliveryStatusWebhook mem-parsing body event webhook "message.status"
// Kirimdev (amplop Meta yang sama dengan message.received, cuma field
// "statuses" bukan "messages", §12 techspec) jadi daftar DeliveryStatus.
// Pemanggil membedakan event ini dari message.received lewat header
// "X-Kirim-Event", BUKAN dari isi body (lihat KirimdevWebhook handler).
func (c *Client) ParseDeliveryStatusWebhook(body []byte) ([]DeliveryStatus, error) {
	var envelope webhookEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	var out []DeliveryStatus
	for _, entry := range envelope.Entry {
		for _, change := range entry.Changes {
			for _, st := range change.Value.Statuses {
				ds := DeliveryStatus{WAMessageID: st.ID, Status: st.Status}
				if len(st.Errors) > 0 {
					e := st.Errors[0]
					ds.ErrorDetail = fmt.Sprintf("%s: %s (code %d)", e.Title, e.Message, e.Code)
				}
				out = append(out, ds)
			}
		}
	}
	return out, nil
}

// kirimdevEventEnvelope adalah amplop KIRIMDEV-NATIVE (bukan format Meta) —
// dipakai event selain message.received/message.status, mis. message.sent
// (docs.kirimdev.com/webhooks/events/).
type kirimdevEventEnvelope struct {
	Type string `json:"type"`
	Data struct {
		Message struct {
			ID         string `json:"id"`          // ID internal Kirimdev, "msg_..." — sama dengan SendResult.MessageID
			ProviderID string `json:"provider_id"` // wamid ASLI Meta, muncul pertama kali di sini
		} `json:"message"`
	} `json:"data"`
}

// ParseMessageSentWebhook mem-parsing event "message.sent" (amplop
// Kirimdev-native, BUKAN format Meta) — event inilah satu-satunya sumber
// wamid ASLI untuk pesan yang kita kirim sendiri (respons sinkron
// SendTemplate cuma punya ID internal Kirimdev, lihat sendTemplateResponse).
// kirimdevMessageID dipakai mencari baris tr_whatsapp_message_log yang mau
// diperbarui (WHERE waMessageID = kirimdevMessageID), wamid jadi nilai
// barunya — supaya context.id balasan PIC (yang selalu wamid asli) nanti
// bisa dicocokkan (§1a.5 techspec).
func (c *Client) ParseMessageSentWebhook(body []byte) (kirimdevMessageID, wamid string, err error) {
	var event kirimdevEventEnvelope
	if err := json.Unmarshal(body, &event); err != nil {
		return "", "", err
	}
	return event.Data.Message.ID, event.Data.Message.ProviderID, nil
}
