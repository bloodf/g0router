package rtk

type CavemanLevel string

const (
	CavemanLite  CavemanLevel = "lite"
	CavemanFull  CavemanLevel = "full"
	CavemanUltra CavemanLevel = "ultra"
)

const cavemanSharedBoundaries = "Code blocks, file paths, commands, errors, URLs: keep exact. Security warnings, irreversible action confirmations, multi-step ordered sequences: write normal. Resume terse style after."

const CavemanLitePrompt = "Respond tersely. Keep grammar and full sentences but drop filler, hedging and pleasantries. Pattern: state the thing, the action, the reason. Then next step. " + cavemanSharedBoundaries + " Active every response until user asks for normal mode."

const CavemanFullPrompt = "Respond like terse caveman. All technical substance stays exact, only fluff dies. Drop articles, filler, pleasantries, hedging. Fragments OK. Use short synonyms. Pattern: thing action reason. Next step. " + cavemanSharedBoundaries + " Active every response until user asks for normal mode."

const CavemanUltraPrompt = "Respond ultra-terse. Maximum compression. Telegraphic. Abbreviate common technical terms, strip conjunctions, use arrows for causality. One word when one word enough. Pattern: thing -> result. Fix. " + cavemanSharedBoundaries + " Active every response until user asks for normal mode."

func cavemanPrompt(level CavemanLevel) (string, bool) {
	switch level {
	case CavemanLite:
		return CavemanLitePrompt, true
	case CavemanFull:
		return CavemanFullPrompt, true
	case CavemanUltra:
		return CavemanUltraPrompt, true
	default:
		return "", false
	}
}
