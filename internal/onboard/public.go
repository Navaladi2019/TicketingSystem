package onboard

// publicDomains are public email providers. No tenant may claim one: the
// first claim would win it for good (domain ownership is first-claim-wins),
// and the real owners of those addresses are not the tenant.
//
//nolint:gochecknoglobals // read-only allow-list
var publicDomains = map[string]bool{
	"gmail.com": true, "googlemail.com": true, "outlook.com": true, "hotmail.com": true, "live.com": true,
	"msn.com": true, "yahoo.com": true, "ymail.com": true, "aol.com": true, "icloud.com": true, "me.com": true,
	"mac.com": true, "proton.me": true, "protonmail.com": true, "pm.me": true, "gmx.com": true, "gmx.net": true,
	"mail.com": true, "zoho.com": true, "yandex.com": true, "yandex.ru": true, "mail.ru": true, "qq.com": true,
	"163.com": true, "126.com": true, "fastmail.com": true, "tutanota.com": true, "hey.com": true,
	"yahoo.co.uk": true, "yahoo.co.in": true, "hotmail.co.uk": true, "live.co.uk": true, "web.de": true,
}

// publicEmailDomain reports whether d is a listed public email provider.
func publicEmailDomain(d string) bool { return publicDomains[d] }
