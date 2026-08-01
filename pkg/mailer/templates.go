package mailer

// Template email disesuaikan dengan brand FSLDK Indonesia (hijau #00933b).
// Markup berbasis <table> dengan CSS inline agar tampil konsisten di
// berbagai klien email maupun renderer webkit (wkhtmltopdf/wkhtmltoimage).
// Logo disematkan sebagai lampiran inline (cid) — lihat mailer.go.

const baseWrapperHead = `<!doctype html>
<html lang="id"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;background:#f2f3f2;font-family:Arial,Helvetica,sans-serif;color:#14171a;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f2f3f2;padding:24px 0;">
<tr><td align="center">
<table role="presentation" width="560" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:16px;overflow:hidden;max-width:560px;width:100%;">
<tr><td style="background:#00933b;padding:24px 32px;">
<table role="presentation" cellpadding="0" cellspacing="0"><tr>
<td style="padding-right:12px;"><img src="cid:{{.LogoCID}}" alt="FSLDK Indonesia" width="40" height="40" style="display:block;border-radius:8px;"></td>
<td>
<span style="color:#ffffff;font-size:20px;font-weight:800;letter-spacing:.3px;">FSLDK Indonesia</span>
<div style="color:#eafbf1;font-size:12px;margin-top:2px;">Forum Silaturahmi Lembaga Dakwah Kampus</div>
</td>
</tr></table>
</td></tr>
<tr><td style="padding:32px;">`

const baseWrapperFoot = `</td></tr>
<tr><td style="padding:20px 32px;background:#fbfaf7;color:#9aa1a8;font-size:12px;line-height:1.6;">
Email ini dikirim otomatis oleh sistem FSLDK Indonesia. Jika Anda tidak merasa melakukan permintaan ini, abaikan email ini.<br>
&copy; FSLDK Indonesia — Menyatukan Langkah Dakwah Kampus se-Indonesia.
</td></tr>
</table></td></tr></table></body></html>`

const verificationTemplate = baseWrapperHead + `
<h1 style="font-size:22px;margin:0 0 12px;">Verifikasi Email Anda</h1>
<p style="font-size:15px;line-height:1.7;color:#5b6168;margin:0 0 20px;">
Assalamu'alaikum {{.Name}},<br><br>
Terima kasih telah mendaftar di website FSLDK Indonesia. Silakan klik tombol di bawah untuk memverifikasi alamat email Anda. Tautan ini berlaku selama 60 menit.
</p>
<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="background:#00933b;border-radius:10px;">
<a href="{{.URL}}" style="display:inline-block;color:#ffffff;text-decoration:none;font-weight:700;padding:14px 28px;font-size:15px;">Verifikasi Email</a>
</td></tr></table>
<p style="font-size:13px;color:#9aa1a8;margin:24px 0 0;line-height:1.6;">
Jika tombol tidak berfungsi, salin tautan berikut ke peramban Anda:<br>
<span style="color:#007a31;word-break:break-all;">{{.URL}}</span>
</p>
` + baseWrapperFoot

const passwordResetTemplate = baseWrapperHead + `
<h1 style="font-size:22px;margin:0 0 12px;">Atur Ulang Kata Sandi</h1>
<p style="font-size:15px;line-height:1.7;color:#5b6168;margin:0 0 20px;">
Assalamu'alaikum {{.Name}},<br><br>
Kami menerima permintaan untuk mengatur ulang kata sandi akun Anda. Klik tombol di bawah untuk membuat kata sandi baru. Tautan ini berlaku selama 60 menit.
</p>
<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="background:#00933b;border-radius:10px;">
<a href="{{.URL}}" style="display:inline-block;color:#ffffff;text-decoration:none;font-weight:700;padding:14px 28px;font-size:15px;">Atur Ulang Kata Sandi</a>
</td></tr></table>
<p style="font-size:13px;color:#9aa1a8;margin:24px 0 0;line-height:1.6;">
Jika Anda tidak meminta ini, abaikan email ini dan kata sandi Anda tetap aman.<br>
Tautan: <span style="color:#007a31;word-break:break-all;">{{.URL}}</span>
</p>
` + baseWrapperFoot
