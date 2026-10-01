package v1

import "html"

// Reserve room for the existing lead fields and Telegram's HTML envelope.
// The original notes remain unchanged in CRM.
func telegramLeadComment(value string) string {
	const limit = 1500
	units := 0
	for index, r := range value {
		units += telegramHTMLUnits(html.EscapeString(string(r)))
		if units > limit {
			return value[:index] + "… (полный комментарий — в CRM)"
		}
	}
	return value
}
