package redact

import "regexp"

type Rule struct {
	Name    string
	Pattern *regexp.Regexp
	Replace string
}

type Result struct {
	Text    string
	Applied bool
	Rules   []string
}

type Redactor struct {
	Rules []Rule
}

func Default() Redactor {
	return Redactor{Rules: []Rule{
		{"anthropic_key", regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]+`), `sk-ant-[REDACTED]`},
		{"openai_key", regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`), `sk-[REDACTED]`},
		{"github_token", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{20,}`), `gh[token]_[REDACTED]`},
		{"slack_token", regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]+`), `xox-[REDACTED]`},
		{"aws_access_key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`), `[REDACTED:aws_access_key]`},
		{"bearer_token", regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]{16,}`), `Bearer [REDACTED]`},
		{"basic_auth", regexp.MustCompile(`(?i)Basic\s+[A-Za-z0-9+/=]{16,}`), `Basic [REDACTED]`},
		{"private_key", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`), `[REDACTED:private_key]`},
		{"env_secret", regexp.MustCompile(`(?im)^([A-Z0-9_]*(?:API_KEY|TOKEN|SECRET|PASSWORD|PRIVATE_KEY)[A-Z0-9_]*\s*=\s*).+$`), `$1[REDACTED]`},
		{"url_credential", regexp.MustCompile(`https?://([^\s:/@]+):([^\s/@]+)@`), `https://[REDACTED]@`},
	}}
}

func (r Redactor) Redact(text string) Result {
	res := Result{Text: text}
	for _, rule := range r.Rules {
		if rule.Pattern.MatchString(res.Text) {
			res.Text = rule.Pattern.ReplaceAllString(res.Text, rule.Replace)
			res.Applied = true
			res.Rules = append(res.Rules, rule.Name)
		}
	}
	return res
}
