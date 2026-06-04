package oauth

import (
	"context"
	"errors"
)

const (
	qianfanProviderID ProviderID = "qianfan"
	qianfanAuthURL               = "https://console.bce.baidu.com/iam/#/iam/apikey/list"
)

// QianfanFlow implements Qianfan direct API-key credential capture.
type QianfanFlow struct {
	apiKey *apiKeyFlow
}

// NewQianfanFlow returns a Qianfan API-key compatible OAuth flow.
func NewQianfanFlow() *QianfanFlow {
	return &QianfanFlow{
		apiKey: newAPIKeyFlow(qianfanProviderID, qianfanAuthURL),
	}
}

func (f *QianfanFlow) ProviderID() ProviderID {
	return qianfanProviderID
}

func (f *QianfanFlow) Start(ctx context.Context) (AuthSession, error) {
	return f.apiKey.start(ctx)
}

func (f *QianfanFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return f.apiKey.exchange(ctx, session, code)
}

func (f *QianfanFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("qianfan api key flow does not support poll")
}
