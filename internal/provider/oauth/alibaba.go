package oauth

import (
	"context"
	"errors"
)

const (
	alibabaProviderID ProviderID = "alibaba"
	alibabaAuthURL               = "https://bailian.console.aliyun.com/?tab=model#/api-key"
)

// AlibabaFlow implements Alibaba direct API-key credential capture.
type AlibabaFlow struct {
	apiKey *apiKeyFlow
}

// NewAlibabaFlow returns an Alibaba API-key compatible OAuth flow.
func NewAlibabaFlow() *AlibabaFlow {
	return &AlibabaFlow{
		apiKey: newAPIKeyFlow(alibabaProviderID, alibabaAuthURL),
	}
}

func (f *AlibabaFlow) ProviderID() ProviderID {
	return alibabaProviderID
}

func (f *AlibabaFlow) Start(ctx context.Context) (AuthSession, error) {
	return f.apiKey.start(ctx)
}

func (f *AlibabaFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return f.apiKey.exchange(ctx, session, code)
}

func (f *AlibabaFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("alibaba api key flow does not support poll")
}
