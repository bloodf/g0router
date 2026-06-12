package admin

import (
	"encoding/json"
	"fmt"

	"github.com/valyala/fasthttp"
)

// supportedLocales is the set of dashboard locales mirrored from 9router's
// src/i18n/config.js (frozen @ 827e5c3). The source list contains 33 entries.
var supportedLocales = map[string]struct{}{
	"en":    {},
	"vi":    {},
	"zh-CN": {},
	"zh-TW": {},
	"ja":    {},
	"pt-BR": {},
	"pt-PT": {},
	"ko":    {},
	"es":    {},
	"de":    {},
	"fr":    {},
	"he":    {},
	"ar":    {},
	"ru":    {},
	"pl":    {},
	"cs":    {},
	"nl":    {},
	"tr":    {},
	"uk":    {},
	"tl":    {},
	"id":    {},
	"th":    {},
	"hi":    {},
	"bn":    {},
	"ur":    {},
	"ro":    {},
	"sv":    {},
	"it":    {},
	"el":    {},
	"hu":    {},
	"fi":    {},
	"da":    {},
	"no":    {},
}

// PostLocale handles POST /api/locale. It validates the requested locale code,
// persists it as a non-HttpOnly cookie so the UI can read the preference, and
// returns the chosen locale in the standard envelope.
func (h *Handlers) PostLocale(ctx *fasthttp.RequestCtx) {
	var body struct {
		Locale string `json:"locale"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &body); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}

	if _, ok := supportedLocales[body.Locale]; !ok {
		writeError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("unsupported locale: %s", body.Locale))
		return
	}

	// Set the locale cookie manually so the attribute names match the spec
	// exactly (Path=/; SameSite=Lax) and the cookie remains readable by JS.
	ctx.Response.Header.Set("Set-Cookie",
		fmt.Sprintf("locale=%s; Path=/; SameSite=Lax", body.Locale))

	writeData(ctx, fasthttp.StatusOK, map[string]string{"locale": body.Locale})
}
