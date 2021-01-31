package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

func requestVerified(secret, method, path string, query map[string][]string, body []byte) bool {
	params := map[string][]string{}
	suppliedMAC := ""

	for key, value := range query {
		if key == "auth_signature" {
			suppliedMAC = value[0]
		} else if key == "body_md5" {
			// ignore
			continue
		} else {
			params[key] = value
		}
	}

	if body != nil && len(body) > 0 {
		hash := md5.Sum(body)
		params["body_md5"] = []string{hex.EncodeToString(hash[:])}
	}

	i := 0
	sortedKeys := make([]string, len(params))
	for key := range params {
		sortedKeys[i] = key
		i++
	}
	sort.Strings(sortedKeys)

	var sb strings.Builder

	for i, key := range sortedKeys {
		if i > 0 {
			sb.WriteRune('&')
		}
		sb.WriteString(key)
		sb.WriteRune('=')
		for ii, value := range params[key] {
			if ii > 0 {
				sb.WriteRune(',')
			}
			sb.WriteString(value)
		}
	}

	signature := method + "\n" + path + "\n" + sb.String()

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signature))
	computedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(suppliedMAC), []byte(computedMAC))
}
