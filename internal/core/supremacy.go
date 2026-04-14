package core

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type SecurityScanner struct{}

type Vulnerability struct {
	Severity  string
	Type      string
	Location  string
	Description string
	Recommendation string
}

func NewSecurityScanner() *SecurityScanner {
	return &SecurityScanner{}
}

func (ss *SecurityScanner) Scan(code string, language string) []*Vulnerability {
	var vulns []*Vulnerability

	vulns = append(vulns, ss.checkInjection(code)...)
	vulns = append(vulns, ss.checkAuthentication(code)...)
	vulns = append(vulns, ss.checkSecrets(code)...)
	vulns = append(vulns, ss.checkCrypto(code)...)
	vulns = append(vulns, ss.checkInputValidation(code)...)

	return vulns
}

func (ss *SecurityScanner) checkInjection(code string) []*Vulnerability {
	var vulns []*Vulnerability

	injectionPatterns := map[string]string{
		`(?i)(sql|squery).*\+`:            "SQL Injection",
		`exec\s*\(.*\+`:                   "Command Injection",
		`eval\s*\(`:                       "Code Injection",
		`innerHTML\s*=.*\+`:               "XSS",
		`dangerouslySetInnerHTML`:         "XSS React",
		`document\.write`:                 "XSS",
	}

	for pattern, vulnType := range injectionPatterns {
		if matched, _ := regexp.MatchString(pattern, code); matched {
			vulns = append(vulns, &Vulnerability{
				Severity:     "HIGH",
				Type:         vulnType,
				Location:     "Code contains potentially unsafe pattern",
				Description:  fmt.Sprintf("Potential %s vulnerability detected", vulnType),
				Recommendation: "Use parameterized queries / sanitization",
			})
		}
	}

	return vulns
}

func (ss *SecurityScanner) checkAuthentication(code string) []*Vulnerability {
	var vulns []*Vulnerability

	authPatterns := map[string]string{
		`password\s*=\s*".*"`:                    "Hardcoded Password",
		`api[_-]?key\s*=\s*".*"`:                  "Hardcoded API Key",
		`secret\s*=\s*".*"`:                       "Hardcoded Secret",
		`auth.*bypass`:                            "Auth Bypass",
		`md5\s*\(.*password`:                      "Weak Hashing",
		`sha1\s*\(.*password`:                     "Weak Hashing",
	}

	for pattern, vulnType := range authPatterns {
		if matched, _ := regexp.MatchString(pattern, code); matched {
			severity := "HIGH"
			if vulnType == "Weak Hashing" {
				severity = "MEDIUM"
			}
			vulns = append(vulns, &Vulnerability{
				Severity:     severity,
				Type:         vulnType,
				Location:     "Code contains credential pattern",
				Description:  vulnType + " found in code",
				Recommendation: "Use environment variables / secure vault",
			})
		}
	}

	return vulns
}

func (ss *SecurityScanner) checkSecrets(code string) []*Vulnerability {
	var vulns []*Vulnerability

	secretPatterns := []string{
		`aws_access_key`,
		`aws_secret_key`,
		`private_key`,
		`bearer\s+[A-Za-z0-9_-]{20,}`,
		`token["\s]*[:=]["\s]*[A-Za-z0-9_-]{20,}`,
	}

	for _, pattern := range secretPatterns {
		if matched, _ := regexp.MatchString(pattern, code); matched {
			vulns = append(vulns, &Vulnerability{
				Severity:     "CRITICAL",
				Type:         "Secret Exposure",
				Location:     "Hardcoded secret detected",
				Description:  "Credentials found in source code",
				Recommendation: "Use secrets manager / environment variables",
			})
		}
	}

	return vulns
}

func (ss *SecurityScanner) checkCrypto(code string) []*Vulnerability {
	var vulns []*Vulnerability

	cryptoPatterns := map[string]string{
		`crypto\.createCipher`:               "Weak Crypto (createCipher deprecated)",
		`DES\.encrypt`:                       "Weak Crypto (DES deprecated)",
		`RijndaelManaged`:                    "Weak Crypto (non-standard)",
		`random\.Math\.random`:               "Weak Random",
	}

	for pattern, vulnType := range cryptoPatterns {
		if matched, _ := regexp.MatchString(pattern, code); matched {
			vulns = append(vulns, &Vulnerability{
				Severity:     "MEDIUM",
				Type:         vulnType,
				Location:     "Cryptographic weakness",
				Description:  vulnType,
				Recommendation: "Use AES-256-GCM / crypto.randomBytes",
			})
		}
	}

	return vulns
}

func (ss *SecurityScanner) checkInputValidation(code string) []*Vulnerability {
	var vulns []*Vulnerability

	if matched, _ := regexp.MatchString(`(?i)(req\.params|request\.param)\s*\.\s*[^=]+\s*=`, code); matched {
		vulns = append(vulns, &Vulnerability{
			Severity:     "MEDIUM",
			Type:         "Unvalidated Input",
			Location:     "Request parameter used directly",
			Description:  "User input not validated before use",
			Recommendation: "Always validate and sanitize user input",
		})
	}

	return vulns
}

func (ss *SecurityScanner) GenerateSafeCode(template string, language string) string {
	safePatterns := map[string]string{
		"go":     `fmt.Sprintf("%s", userInput)`,
		"python": `html.escape(userInput)`,
		"js":     `DOMPurify.sanitize(userInput)`,
	}

	if safe, ok := safePatterns[language]; ok {
		return safe
	}
	return template
}

type CodeTranslator struct{}

func NewCodeTranslator() *CodeTranslator {
	return &CodeTranslator{}
}

func (ct *CodeTranslator) Translate(code, fromLang, toLang string) (string, error) {
	prompt := fmt.Sprintf(`Translate this %s code to %s. Maintain all functionality and add comments explaining the translation.

%s

Return only the translated code with no explanations.`, fromLang, toLang, code)

	return prompt, nil
}

type DependencyGhost struct{}

func NewDependencyGhost() *DependencyGhost {
	return &DependencyGhost{}
}

func (dg *DependencyGhost) Install(projectType, projectPath string) (string, error) {
	var cmd string

	switch projectType {
	case "node":
		cmd = "npm install"
	case "go":
		cmd = "go mod download"
	case "python":
		cmd = "pip install -r requirements.txt"
	case "rust":
		cmd = "cargo fetch"
	case "android":
		cmd = "./gradlew dependencies --refresh-dependencies"
	case "flutter":
		cmd = "flutter pub get"
	default:
		return "", fmt.Errorf("unknown project type: %s", projectType)
	}

	execCmd := exec.Command("sh", "-c", cmd)
	execCmd.Dir = projectPath
	output, err := execCmd.CombinedOutput()

	if err != nil {
		return string(output), err
	}

	return fmt.Sprintf("✓ Dependencies installed for %s project", projectType), nil
}

type WikiGenerator struct{}

func NewWikiGenerator() *WikiGenerator {
	return &WikiGenerator{}
}

func (wg *WikiGenerator) GenerateDocumentation(projectPath string) (string, error) {
	var doc strings.Builder

	doc.WriteString("# Project Documentation\n\n")
	doc.WriteString("Auto-generated by SIBY-AGENTIQ\n\n")

	doc.WriteString("## Table of Contents\n")
	doc.WriteString("- [Overview](#overview)\n")
	doc.WriteString("- [Installation](#installation)\n")
	doc.WriteString("- [Usage](#usage)\n")
	doc.WriteString("- [API Reference](#api-reference)\n")
	doc.WriteString("- [Architecture](#architecture)\n\n")

	doc.WriteString("## Overview\n")
	doc.WriteString("Project generated with AI assistance.\n\n")

	doc.WriteString("## Installation\n```bash\n")
	doc.WriteString("# Add installation commands\n")
	doc.WriteString("```\n\n")

	return doc.String(), nil
}
