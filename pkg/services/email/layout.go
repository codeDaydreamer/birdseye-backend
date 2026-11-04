package email

// The email layout shared (as a copy) by Birdseye, Dynapay and Academix: the product's own colours and
// name at the top, one message in a white card, and the 816 Dynamics footer. Table-based with inline
// styles so it renders the same in Gmail, Outlook and phone mail apps.

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

// Brand is how one product looks in email
type Brand struct {
	Product string // "Dynapay"
	Color   string // buttons, badge, top bar
	Dark    string // code text, links
	Tint    string // code box background
	URL     string // where the product name links to
}

// Button is the main call to action
type Button struct {
	Label string
	URL   string
}

// Detail is one label/value row, e.g. "Amount" / "KES 1,500"
type Detail struct {
	Label string
	Value string
}

// Message is one email. Lines and Note are HTML: escape anything a user typed with Esc.
type Message struct {
	Preview  string   // one line shown next to the subject in the inbox
	Title    string   // heading inside the card
	Greeting string   // "Hi Jane," (plain text)
	Lines    []string // paragraphs
	Code     string   // a one-time code shown large
	Details  []Detail
	Button   *Button
	Note     string // small print under the button
	Danger   bool   // red top bar and button, for failures
}

// Esc escapes user-entered text for use inside Lines or Note
func Esc(s string) string { return html.EscapeString(s) }

const font = `-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif`

// Render returns the HTML and plain-text bodies of a message
func Render(b Brand, m Message) (string, string) {
	color := b.Color
	if m.Danger {
		color = "#dc2626"
	}
	var w strings.Builder
	p := func(format string, a ...interface{}) { fmt.Fprintf(&w, format, a...) }

	p(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"><title>%s</title></head>`, Esc(m.Title))
	p(`<body style="margin:0;padding:0;background:#f3f4f6;">`)
	if m.Preview != "" {
		p(`<div style="display:none;max-height:0;overflow:hidden;opacity:0;">%s</div>`, Esc(m.Preview))
	}
	p(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f3f4f6;"><tr><td align="center" style="padding:32px 16px;">`)
	p(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="max-width:560px;">`)

	// Product name
	p(`<tr><td style="padding:0 4px 16px;font-family:%s;"><a href="%s" style="text-decoration:none;color:#111827;">`, font, Esc(b.URL))
	p(`<span style="display:inline-block;width:28px;height:28px;line-height:28px;border-radius:7px;background:%s;color:#ffffff;font-weight:700;font-size:15px;text-align:center;vertical-align:middle;">%s</span>`, b.Color, Esc(b.Product[:1]))
	p(`<span style="font-size:17px;font-weight:700;vertical-align:middle;padding-left:8px;color:#111827;">%s</span></a></td></tr>`, Esc(b.Product))

	// Card
	p(`<tr><td style="background:#ffffff;border-radius:12px;border-top:4px solid %s;padding:32px 32px 28px;font-family:%s;color:#374151;font-size:15px;line-height:1.6;">`, color, font)
	p(`<h1 style="margin:0 0 16px;font-size:22px;line-height:1.3;color:#111827;font-weight:700;">%s</h1>`, Esc(m.Title))
	if m.Greeting != "" {
		p(`<p style="margin:0 0 14px;">%s</p>`, Esc(m.Greeting))
	}
	for _, l := range m.Lines {
		p(`<p style="margin:0 0 14px;">%s</p>`, l)
	}
	if m.Code != "" {
		p(`<div style="margin:20px 0;padding:18px;border-radius:10px;background:%s;text-align:center;font-family:'SFMono-Regular',Menlo,Consolas,monospace;font-size:30px;font-weight:700;letter-spacing:8px;color:%s;">%s</div>`, b.Tint, b.Dark, Esc(m.Code))
	}
	if len(m.Details) > 0 {
		p(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="margin:18px 0;border:1px solid #e5e7eb;border-radius:10px;">`)
		for i, d := range m.Details {
			border := "border-top:1px solid #e5e7eb;"
			if i == 0 {
				border = ""
			}
			p(`<tr><td style="%spadding:10px 14px;color:#6b7280;font-size:14px;">%s</td><td align="right" style="%spadding:10px 14px;color:#111827;font-size:14px;font-weight:600;">%s</td></tr>`, border, Esc(d.Label), border, Esc(d.Value))
		}
		p(`</table>`)
	}
	if m.Button != nil {
		p(`<table role="presentation" cellpadding="0" cellspacing="0" style="margin:24px 0 8px;"><tr><td style="border-radius:8px;background:%s;">`, color)
		p(`<a href="%s" style="display:inline-block;padding:12px 22px;font-family:%s;font-size:15px;font-weight:600;color:#ffffff;text-decoration:none;border-radius:8px;">%s</a></td></tr></table>`, Esc(m.Button.URL), font, Esc(m.Button.Label))
		p(`<p style="margin:12px 0 0;font-size:12px;color:#6b7280;word-break:break-all;">If the button doesn't work, open this link: <a href="%s" style="color:%s;">%s</a></p>`, Esc(m.Button.URL), b.Dark, Esc(m.Button.URL))
	}
	if m.Note != "" {
		p(`<p style="margin:20px 0 0;padding-top:16px;border-top:1px solid #f3f4f6;font-size:13px;color:#6b7280;">%s</p>`, m.Note)
	}
	p(`</td></tr>`)

	// 816 Dynamics footer
	p(`<tr><td align="center" style="padding:24px 8px 0;font-family:%s;font-size:12px;line-height:1.7;color:#6b7280;">`, font)
	p(`%s is made by <a href="https://www.816-dynamics.com" style="color:#374151;font-weight:600;text-decoration:none;">816 Dynamics</a>.<br>`, Esc(b.Product))
	p(`Need help? <a href="mailto:hello@816-dynamics.com" style="color:#6b7280;">hello@816-dynamics.com</a> &middot; <a href="https://wa.me/254792803063" style="color:#6b7280;">WhatsApp +254 792 803 063</a><br>`)
	p(`&copy; %d 816 Dynamics</td></tr>`, time.Now().Year())
	p(`</table></td></tr></table></body></html>`)

	return w.String(), text(b, m)
}

var tags = regexp.MustCompile(`<[^>]+>`)

func plain(s string) string { return html.UnescapeString(tags.ReplaceAllString(s, "")) }

func text(b Brand, m Message) string {
	var w strings.Builder
	w.WriteString(b.Product + "\n\n" + m.Title + "\n\n")
	if m.Greeting != "" {
		w.WriteString(m.Greeting + "\n\n")
	}
	for _, l := range m.Lines {
		w.WriteString(plain(l) + "\n\n")
	}
	if m.Code != "" {
		w.WriteString("    " + m.Code + "\n\n")
	}
	for _, d := range m.Details {
		w.WriteString(d.Label + ": " + d.Value + "\n")
	}
	if len(m.Details) > 0 {
		w.WriteString("\n")
	}
	if m.Button != nil {
		w.WriteString(m.Button.Label + ": " + m.Button.URL + "\n\n")
	}
	if m.Note != "" {
		w.WriteString(plain(m.Note) + "\n\n")
	}
	w.WriteString("--\n" + b.Product + " is made by 816 Dynamics (https://www.816-dynamics.com)\nHelp: hello@816-dynamics.com, WhatsApp +254 792 803 063\n")
	return w.String()
}
